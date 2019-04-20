package cron

import (
	"context"
	"fmt"
	"os"

	"github.com/icco/cron/goodreads"
	"github.com/icco/cron/pinboard"
	"github.com/icco/cron/tweets"
)

func Act(ctx context.Context, job string) error {
	gqlToken := os.Getenv("GQL_TOKEN")
	if gqlToken == "" {
		return fmt.Errorf("GQL_TOKEN is unset")
	}

	twitterAuth := &tweets.TwitterAuth{
		ConsumerKey:    os.Getenv("TWITTER_CONSUMER_KEY"),
		ConsumerSecret: os.Getenv("TWITTER_CONSUMER_SECRET"),
		AccessToken:    os.Getenv("TWITTER_ACCESS_TOKEN"),
		AccessSecret:   os.Getenv("TWITTER_ACCESS_SECRET"),
	}

	pinboardToken := os.Getenv("PINBOARD_TOKEN")
	if pinboardToken == "" {
		return fmt.Errorf("PINBOARD_TOKEN is unset")
	}

	goodreadsToken := os.Getenv("GOODREADS_TOKEN")
	if goodreadsToken == "" {
		return fmt.Errorf("GOODREADS_TOKEN is unset")
	}

	switch job {
	case "user-tweets":
		err := tweets.SaveUserTweets(ctx, log, gqlToken, twitterAuth)
		if err != nil {
			return err
		}
	case "pinboard":
		err := pinboard.UpdatePins(ctx, log, pinboardToken, gqlToken)
		if err != nil {
			return err
		}
	case "random-tweets":
		err := tweets.CacheRandomTweets(ctx, log, gqlToken, twitterAuth)
		if err != nil {
			return err
		}
	case "goodreads":
		g := &goodreads.Goodreads{
			Log:          log,
			Token:        goodreadsToken,
			GraphQLToken: gqlToken,
		}
		err := g.UpsertBooks(ctx)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown job type: %s", job)
	}

	return nil
}
