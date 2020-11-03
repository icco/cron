package uptime

import (
	"context"
	"fmt"

	monitoring "cloud.google.com/go/monitoring/apiv3"
)

func UpdateServices(ctx context.Context, c *Config) error {
	client, err := monitoring.NewServiceMonitoringClient(ctx)
	if err != nil {
		return fmt.Errorf("service monitoring: %w", err)
	}

	// TODO: Use client.
	_ = client

	return nil
}
