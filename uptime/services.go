package uptime

import (
	"context"
	"fmt"
	"strings"
	"time"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/icco/cron/sites"
	"github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/protobuf/types/known/durationpb"
)

func UpdateServices(ctx context.Context, c *Config) error {
	client, err := monitoring.NewServiceMonitoringClient(ctx)
	if err != nil {
		return fmt.Errorf("service monitoring: %w", err)
	}

	req := &monitoringpb.ListServicesRequest{
		Parent: "projects/" + c.ProjectID,
	}

	svcs := []*monitoringpb.Service{}
	it := client.ListServices(ctx, req)
	for {
		svc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("list services: %w", err)
		}

		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": svc}).Debug("found service")
		svcs = append(svcs, svc)
	}

	for _, s := range sites.All {
		wanted := &monitoringpb.Service{
			DisplayName: s.Deployment,
			Identifier:  &monitoringpb.Service_Custom_{},
			Telemetry: &monitoringpb.Service_Telemetry{
				ResourceName: fmt.Sprintf("//container.googleapis.com/projects/%s/locations/us-central1/clusters/nat-cluster-2/k8s/namespaces/default/services/%s-service", c.ProjectID, s.Deployment),
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
			c.Log.WithFields(logrus.Fields{"job": "uptime", "service": resp}).Debug("created service")
			wanted = resp
		} else {
			req := &monitoringpb.UpdateServiceRequest{
				Service: wanted,
			}
			resp, err := client.UpdateService(ctx, req)
			if err != nil {
				return fmt.Errorf("update service: %w", err)
			}
			c.Log.WithFields(logrus.Fields{"job": "uptime", "service": resp}).Debug("updated service")
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
			c.Log.WithField("policy", policy).Debug("found alert policy")
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
				DisplayName: "slo burn",
				Condition: &monitoringpb.AlertPolicy_Condition_ConditionThreshold{
					ConditionThreshold: &monitoringpb.AlertPolicy_Condition_MetricThreshold{
						Filter:         fmt.Sprintf("select_slo_burn_rate(%q, %q)", sloID, "3600s"),
						ThresholdValue: 10,
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
	}

	if existing != nil {
		wanted.Name = existing.Name
		if len(existing.Conditions) == len(wanted.Conditions) {
			for i, c := range existing.Conditions {
				wanted.Conditions[i].Name = c.Name
			}
		}

		if _, err := client.UpdateAlertPolicy(ctx, &monitoringpb.UpdateAlertPolicyRequest{AlertPolicy: wanted}); err != nil {
			return err
		}
	} else {
		req := &monitoringpb.CreateAlertPolicyRequest{
			Name:        fmt.Sprintf("projects/%s", c.ProjectID),
			AlertPolicy: wanted,
		}
		if _, err := client.CreateAlertPolicy(ctx, req); err != nil {
			return err
		}
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
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": svc, "site": s, "slo": resp}).Debug("found slo")
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
		Goal:        0.99,
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
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": svc, "site": s, "slo": resp}).Debug("updated slo")
		return resp, nil
	} else {
		req := &monitoringpb.CreateServiceLevelObjectiveRequest{
			Parent:                svc.Name,
			ServiceLevelObjective: want,
		}
		resp, err := client.CreateServiceLevelObjective(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("create slo: %w", err)
		}
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": svc, "site": s, "slo": resp}).Debug("created slo")
		return resp, nil
	}

	return nil, fmt.Errorf("unknown logic error")
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
		name := fmt.Sprintf("%s-service-8080", dep)
		if strings.Contains(b.Name, name) {
			return b.Name, nil
		}
	}

	return "", fmt.Errorf("no backends found")
}
