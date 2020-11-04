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
		exists := false
		for _, svc := range svcs {
			if svc.DisplayName == s.Deployment {
				exists = true
				break
			}
		}

		if !exists {
			req := &monitoringpb.CreateServiceRequest{
				Parent:    "projects/" + c.ProjectID,
				ServiceId: s.Deployment,
				Service: &monitoringpb.Service{
					DisplayName: s.Deployment,
					Identifier:  &monitoringpb.Service_Custom_{},
				},
			}
			if _, err := client.CreateService(ctx, req); err != nil {
				return err
			}
		}
	}

	return nil
}
