package uptime

import (
	"context"
	"fmt"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"github.com/icco/cron/sites"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

// UpdateServices updates monitoring for all of our services.
func UpdateServices(ctx context.Context, c *Config) error {
	client, err := monitoring.NewServiceMonitoringClient(ctx)
	if err != nil {
		return fmt.Errorf("service monitoring: %w", err)
	}

	var svcs []*monitoringpb.Service
	req := &monitoringpb.ListServicesRequest{
		Parent: "projects/" + c.ProjectID,
	}
	it := client.ListServices(ctx, req)
	for {
		svc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("list services: %w", err)
		}

		c.Log.Debugw("found service", "job", "uptime", "service", svc)
		if strings.Contains(svc.Telemetry.ResourceName, "container.googleapis.com") {
			c.Log.Debugw("delete service", "job", "uptime", "service", svc)
			if err := client.DeleteService(ctx, &monitoringpb.DeleteServiceRequest{Name: svc.Name}); err != nil {
				return fmt.Errorf("delete service %q: %w", svc.Name, err)
			}
		} else {
			svcs = append(svcs, svc)
		}
	}

	location := "us-central1"
	for _, s := range sites.All {
		wanted := &monitoringpb.Service{
			DisplayName: s.Deployment,
			Identifier:  &monitoringpb.Service_Custom_{},
			Telemetry: &monitoringpb.Service_Telemetry{
				ResourceName: fmt.Sprintf(
					"//run.googleapis.com/projects/%s/locations/%s/services/%s",
					c.ProjectID,
					location,
					s.Deployment,
				),
			},
		}

		exists := false
		for _, svc := range svcs {
			if svc.DisplayName == s.Deployment {
				exists = true
				wanted.Name = svc.Name
				break
			}
		}

		if !exists {
			req := &monitoringpb.CreateServiceRequest{
				Parent:  "projects/" + c.ProjectID,
				Service: wanted,
			}
			resp, err := client.CreateService(ctx, req)
			if err != nil {
				return fmt.Errorf("create service: %w", err)
			}
			c.Log.Debugw("created service", "job", "uptime", "service", resp)
			wanted = resp
		} else {
			req := &monitoringpb.UpdateServiceRequest{
				Service: wanted,
			}
			resp, err := client.UpdateService(ctx, req)
			if err != nil {
				return fmt.Errorf("update service: %w", err)
			}
			c.Log.Debugw("updated service", "job", "uptime", "service", resp)
			wanted = resp
		}

		slo, err := c.addSLO(ctx, s, wanted)
		if err != nil {
			return fmt.Errorf("add slo: %w", err)
		}

		if err := c.addAlert(ctx, s, slo.Name); err != nil {
			return fmt.Errorf("add alert: %w", err)
		}
	}

	return nil
}

func (c *Config) addAlert(ctx context.Context, s sites.SiteMap, sloID string) error {
	alertType := "slo"
	alertNotification := "projects/icco-cloud/notificationChannels/2074431925909529711"

	client, err := monitoring.NewAlertPolicyClient(ctx)
	if err != nil {
		return fmt.Errorf("alert policy: %w", err)
	}

	var existing *monitoringpb.AlertPolicy
	it := client.ListAlertPolicies(ctx, &monitoringpb.ListAlertPoliciesRequest{
		Name: fmt.Sprintf("projects/%s", c.ProjectID),
	})
	for {
		policy, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("list policies: %w", err)
		}

		if policy.UserLabels["type"] == alertType && policy.UserLabels["service"] == s.Deployment {
			c.Log.Debugw("found alert policy", "policy", policy)
			existing = policy
		}
	}

	wanted := &monitoringpb.AlertPolicy{
		DisplayName:          fmt.Sprintf("SLO Burn Alert %s", s.Host),
		NotificationChannels: []string{alertNotification},
		Combiner:             monitoringpb.AlertPolicy_AND,
		UserLabels: map[string]string{
			"type":    alertType,
			"service": s.Deployment,
		},
		Conditions: []*monitoringpb.AlertPolicy_Condition{
			{
				DisplayName: fmt.Sprintf("SLO Burn for %s", s.Host),
				Condition: &monitoringpb.AlertPolicy_Condition_ConditionThreshold{
					ConditionThreshold: &monitoringpb.AlertPolicy_Condition_MetricThreshold{
						Filter:         fmt.Sprintf("select_slo_burn_rate(%q, %q)", sloID, "3600s"),
						ThresholdValue: 60,
						Trigger: &monitoringpb.AlertPolicy_Condition_Trigger{
							Type: &monitoringpb.AlertPolicy_Condition_Trigger_Count{
								Count: 1,
							},
						},
						Duration:   durationpb.New(time.Minute * 5),
						Comparison: monitoringpb.ComparisonType_COMPARISON_GT,
					},
				},
			},
		},
		Documentation: &monitoringpb.AlertPolicy_Documentation{
			MimeType: "text/markdown",
			Content:  "An SLO alert fires when the number of non 200 http requests increases greatly.",
		},
	}

	if existing != nil {
		wanted.Name = existing.Name
		if len(existing.Conditions) == len(wanted.Conditions) {
			for i, c := range existing.Conditions {
				wanted.Conditions[i].Name = c.Name
			}
		}
		resp, err := client.UpdateAlertPolicy(ctx, &monitoringpb.UpdateAlertPolicyRequest{AlertPolicy: wanted})
		if err != nil {
			return err
		}
		c.Log.Debugw("updated alert policy", "job", "uptime", "site", s, "response", resp)
	} else {
		req := &monitoringpb.CreateAlertPolicyRequest{
			Name:        fmt.Sprintf("projects/%s", c.ProjectID),
			AlertPolicy: wanted,
		}
		resp, err := client.CreateAlertPolicy(ctx, req)
		if err != nil {
			return err
		}
		c.Log.Debugw("created alert policy", "job", "uptime", "site", s, "response", resp)
	}

	return nil
}

func (c *Config) addSLO(ctx context.Context, s sites.SiteMap, svc *monitoringpb.Service) (*monitoringpb.ServiceLevelObjective, error) {
	client, err := monitoring.NewServiceMonitoringClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("service monitoring: %w", err)
	}

	var slo *monitoringpb.ServiceLevelObjective
	it := client.ListServiceLevelObjectives(ctx, &monitoringpb.ListServiceLevelObjectivesRequest{
		Parent: svc.Name,
	})
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list slos: %w", err)
		}
		slo = resp
		c.Log.Debugw("found slo", "job", "uptime", "service", svc, "site", s, "slo", resp)
	}

	metric := "loadbalancing.googleapis.com/https/backend_request_count"
	resource := "https_lb_rule"
	backend, err := c.getBackend(ctx, s.Deployment)
	if err != nil {
		return nil, fmt.Errorf("get backend: %w", err)
	}
	want := &monitoringpb.ServiceLevelObjective{
		DisplayName: fmt.Sprintf("Generated SLO for %s", s.Host),
		Period:      &monitoringpb.ServiceLevelObjective_RollingPeriod{RollingPeriod: &durationpb.Duration{Seconds: 2419200}},
		Goal:        0.9,
		ServiceLevelIndicator: &monitoringpb.ServiceLevelIndicator{
			Type: &monitoringpb.ServiceLevelIndicator_RequestBased{
				RequestBased: &monitoringpb.RequestBasedSli{
					Method: &monitoringpb.RequestBasedSli_GoodTotalRatio{
						GoodTotalRatio: &monitoringpb.TimeSeriesRatio{
							GoodServiceFilter:  fmt.Sprintf("metric.type=%q resource.type=%q resource.labels.backend_target_name=%q metric.labels.response_code_class=\"200\"", metric, resource, backend),
							TotalServiceFilter: fmt.Sprintf("metric.type=%q resource.type=%q resource.labels.backend_target_name=%q", metric, resource, backend),
						},
					},
				},
			},
		},
	}

	if slo != nil {
		want.Name = slo.Name
		req := &monitoringpb.UpdateServiceLevelObjectiveRequest{
			ServiceLevelObjective: want,
		}
		resp, err := client.UpdateServiceLevelObjective(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("update slo: %w", err)
		}
		c.Log.Debugw("updated slo", "job", "uptime", "service", svc, "site", s, "slo", resp)
		return resp, nil
	}

	req := &monitoringpb.CreateServiceLevelObjectiveRequest{
		Parent:                svc.Name,
		ServiceLevelObjective: want,
	}
	resp, err := client.CreateServiceLevelObjective(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create slo: %w", err)
	}

	c.Log.Debugw("created slo", "job", "uptime", "service", svc, "site", s, "slo", resp)
	return resp, nil
}

func (c *Config) getBackend(ctx context.Context, dep string) (string, error) {
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return "", fmt.Errorf("new compute: %w", err)
	}

	list, err := computeService.BackendServices.List(c.ProjectID).Do()
	if err != nil {
		return "", err
	}

	for _, b := range list.Items {
		if strings.Contains(b.Name, name) {
			log.Debugw("found backend", "backend", b, "service", dep)
			return b.Name, nil
		}
	}

	return "", fmt.Errorf("no backends found for %q", dep)
}
