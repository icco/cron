package cron

import (
	"context"
	"fmt"
	"os"

	"github.com/icco/cron/goodreads"
	"github.com/icco/cron/pinboard"
	"github.com/icco/cron/spider"
	"github.com/icco/cron/stats"
	"github.com/icco/cron/tweets"
	"github.com/icco/cron/updater"
	"github.com/icco/cron/uptime"
	"go.opencensus.io/tag"
)

// Act takes a job and calls a sub project to do work.
func Act(octx context.Context, job string) error {
	jobKey, err := tag.NewKey("natwelch.com/keys/job")
	if err != nil {
		log.WithError(err).Warn("could not create oc tag")
	}
	ctx, err := tag.New(octx,
		tag.Upsert(jobKey, job),
	)
	if err != nil {
		log.WithError(err).Warn("could not add oc tag")
	}

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

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN is unset")
	}

	switch job {
	case "test":
		stats.GetAssetMix(ctx)
	case "minute":
		log.Info("> heartbeat")
	case "update-deployments":
		cfg := &updater.Config{
			Log:           log,
			GithubToken:   githubToken,
			GoogleProject: "icco-cloud",
		}
		if err := updater.UpdateWorkspaces(ctx, cfg); err != nil {
			return fmt.Errorf("update workspaces: %w", err)
		}
		if err := updater.UpdateTriggers(ctx, cfg); err != nil {
			return fmt.Errorf("update triggers: %w", err)
		}
	case "spider":
		spider.Crawl(ctx, &spider.Config{Log: log, URL: "https://writing.natwelch.com/"})
	case "user-tweets":
		t := tweets.Twitter{
			TwitterAuth:  twitterAuth,
			Log:          log,
			GraphQLToken: gqlToken,
		}
		err := t.SaveUserTweets(ctx)
		if err != nil {
			return err
		}
	case "pinboard":
		p := &pinboard.Pinboard{
			Token:        pinboardToken,
			Log:          log,
			GraphQLToken: gqlToken,
		}
		err := p.UpdatePins(ctx)
		if err != nil {
			return err
		}
	case "random-tweets":
		t := &tweets.Twitter{
			TwitterAuth:  twitterAuth,
			Log:          log,
			GraphQLToken: gqlToken,
		}
		err := t.CacheRandomTweets(ctx)
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
	case "uptime":
		c := &uptime.Config{
			Log:       log,
			ProjectID: "icco-cloud",
		}

		if err := uptime.UpdateUptimeChecks(ctx, c); err != nil {
			return err
		}

		if err := uptime.UpdateServices(ctx, c); err != nil {
			return err
		}
	case "stats":
		c := &stats.Config{
			Log:          log,
			GraphQLToken: gqlToken,
			OWMKey:       os.Getenv("OPEN_WEATHER_MAP_KEY"),
		}

		if err := c.UpdateOften(ctx); err != nil {
			return err
		}
	case "stats-hourly":
		c := &stats.Config{
			Log:          log,
			GraphQLToken: gqlToken,
		}

		if err := c.UpdateRarely(ctx); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown job type: %s", job)
	}

	return nil
}
