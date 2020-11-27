package uptime

import (
	"context"
	"fmt"
	"strings"

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

		if err := c.addSLO(ctx, s, wanted); err != nil {
			return fmt.Errorf("add slo: %w", err)
		}
	}

	return nil
}

func (c *Config) addSLO(ctx context.Context, s sites.SiteMap, svc *monitoringpb.Service) error {
	client, err := monitoring.NewServiceMonitoringClient(ctx)
	if err != nil {
		return fmt.Errorf("service monitoring: %w", err)
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
			return fmt.Errorf("list slos: %w", err)
		}
		slo = resp
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": svc, "site": s, "slo": resp}).Debug("found slo")
	}

	metric := "loadbalancing.googleapis.com/https/backend_request_count"
	resource := "https_lb_rule"
	backend, err := c.getBackend(ctx, s.Deployment)
	if err != nil {
		return fmt.Errorf("get backend: %w", err)
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
			return fmt.Errorf("update slo: %w", err)
		}
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": svc, "site": s, "slo": resp}).Debug("updated slo")
	} else {
		req := &monitoringpb.CreateServiceLevelObjectiveRequest{
			Parent:                svc.Name,
			ServiceLevelObjective: want,
		}
		resp, err := client.CreateServiceLevelObjective(ctx, req)
		if err != nil {
			return fmt.Errorf("create slo: %w", err)
		}
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": svc, "site": s, "slo": resp}).Debug("created slo")
	}

	return nil
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
