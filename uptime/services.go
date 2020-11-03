package uptime

import (
	"context"

	monitoring "cloud.google.com/go/monitoring/apiv3"
)

func UpdateServices(ctx context.Context, c *Config) error {
	c, err := monitoring.NewServiceMonitoringClient(ctx)
	if err != nil {
		return fmt.Errorf("service monitoring: %w", err)
	}
	// TODO: Use client.
	_ = c

	return nil
}
