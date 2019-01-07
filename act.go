package cron

import (
	"context"
	"fmt"
	"os"

	"github.com/icco/cron/pinboard"
)

func Act(ctx context.Context, job string) error {
	switch job {
	case "hourly":
	case "minute":
	case "five-minute":
	case "fifteen-minute":
		err := pinboard.UpdatePins(ctx, log, os.Getenv("PINBOARD_TOKEN"), os.Getenv("GQL_TOKEN"))
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown job type: %s", job)
	}

	return nil
}
