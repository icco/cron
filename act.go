package cron

import (
	"context"
	"fmt"
	"os"

	"github.com/icco/cron/pinboard"
	"github.com/icco/cron/tweets"
)

func Act(ctx context.Context, job string) error {
	gqlToken := os.Getenv("GQL_TOKEN")

	switch job {
	case "hourly":
	case "minute":
	case "five-minute":
		err := tweets.SaveUserTweets(ctx, log, gqlToken, os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"), os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_SECRET"))
		if err != nil {
			return err
		}
	case "fifteen-minute":
		err := pinboard.UpdatePins(ctx, log, os.Getenv("PINBOARD_TOKEN"), gqlToken)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown job type: %s", job)
	}

	return nil
}
