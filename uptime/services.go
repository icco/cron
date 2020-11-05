package uptime

import (
	"context"
	"fmt"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/icco/cron/sites"
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

		c.Log.Infof("found service: %+v", svc)
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

		req := &monitoringpb.ListServiceLevelObjectivesRequest{
			Parent: wanted.Name,
		}
		it := client.ListServiceLevelObjectives(ctx, req)
		for {
			resp, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			c.Log.Infof("found SLO: %+v", resp)
		}
	}

	return nil
}
