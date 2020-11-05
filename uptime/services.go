package uptime

import (
	"context"
	"fmt"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/icco/cron/sites"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
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
			return err
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
				return err
			}
			wanted.Name = resp.Name
			c.Log.WithFields(logrus.Fields{"job": "uptime", "service": resp}).Debug("created service")
		} else {
			req := &monitoringpb.UpdateServiceRequest{
				Service: wanted,
			}
			resp, err := client.UpdateService(ctx, req)
			if err != nil {
				return err
			}
			c.Log.WithFields(logrus.Fields{"job": "uptime", "service": resp}).Debug("updated service")
		}

		if err := c.addSLO(ctx, s); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) addSLO(ctx context.Context, s sites.Site) error {
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
			return err
		}
		slo = resp
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": s, "slo", resp}).Debug("found slo")
	}

	metric := "loadbalancing.googleapis.com/https/backend_request_count"
	resource := "https_lb_rule"
	backend := fmt.Sprintf("k8s1-dc14e589-default-%s-service-8080-e07b1861", s.Deployment)
	want := &monitoringpb.ServiceLevelObjective{
		DisplayName:   fmt.Sprintf("Generated SLO for %s", s.Host),
		RollingPeriod: {Seconds: 2419200},
		Goal:          0.99,
		ServiceLevelIndicator: {
			RequestBased: {
				GoodTotalRatio: {
					BadServiceFilter:   fmt.Sprintf("metric.type=%q resource.type=%q resource.labels.backend_target_name=%q metric.labels.response_code_class=\"500\"", metric, resource, backend),
					TotalServiceFilter: fmt.Sprintf("metric.type=%q resource.type=%q resource.labels.backend_target_name=%q", metric, resource, backend),
				},
			},
		},
	}

	if slo != nil {
		want.Name = slo.Name
		req := &monitoringpb.UpdateServiceLevelObjectiveRequest{
			ServiceLevelObjective: want,
		}
		resp, err := c.UpdateServiceLevelObjective(ctx, req)
		if err != nil {
			return err
		}
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": s, "slo", resp}).Debug("updated slo")
	} else {
		req := &monitoringpb.CreateServiceLevelObjectiveRequest{
			ServiceLevelObjective: want,
		}
		resp, err := c.CreateServiceLevelObjective(ctx, req)
		if err != nil {
			return err
		}
		c.Log.WithFields(logrus.Fields{"job": "uptime", "service": s, "slo", resp}).Debug("created slo")
	}

	return nil
}
