package cron

import (
	"context"
	"fmt"
	"os"

	"github.com/dgraph-io/ristretto"
	"github.com/icco/cron/code"
	"github.com/icco/cron/goodreads"
	"github.com/icco/cron/pinboard"
	"github.com/icco/cron/spider"
	"github.com/icco/cron/stats"
	"github.com/icco/cron/tweets"
	"github.com/icco/cron/updater"
	"github.com/icco/cron/uptime"
	"go.opencensus.io/tag"
	"go.uber.org/zap"
)

const (
	// GCPProject is the project this runs in.
	GCPProject = "icco-cloud"

	// Service is the name of this service.
	Service = "cron"
)

// Config is our base act config struct.
type Config struct {
	Log   *zap.SugaredLogger
	Cache *ristretto.Cache
}

// Act takes a job and calls a sub project to do work.
func (cfg *Config) Act(octx context.Context, job string) error {
	jobKey, err := tag.NewKey("natwelch.com/keys/job")
	if err != nil {
		cfg.Log.Warnw("could not create oc tag", zap.Error(err))
	}
	ctx, err := tag.New(octx,
		tag.Upsert(jobKey, job),
	)
	if err != nil {
		cfg.Log.Warnw("could not add oc tag", zap.Error(err))
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
		v, err := stats.GetAssetMix(ctx)
		cfg.Log.Warnf("%d, %+v", v, err)
	case "minute":
		cfg.Log.Info("heartbeat")
	case "update-deployments":
		cfg := &updater.Config{
			Log:           cfg.Log,
			GoogleProject: GCPProject,
		}
		if err := cfg.UpdateTriggers(ctx); err != nil {
			return fmt.Errorf("update triggers: %w", err)
		}

		//if err := cfg.UpdateRandomSite(ctx); err != nil {
		//	return fmt.Errorf("update random site: %w", err)
		//}
	case "spider":
		spider.Crawl(ctx, &spider.Config{
			Log: cfg.Log,
			URL: "https://writing.natwelch.com/",
		})
	case "user-tweets":
		t := tweets.Twitter{
			TwitterAuth:  twitterAuth,
			Log:          cfg.Log,
			GraphQLToken: gqlToken,
		}
		err := t.SaveUserTweets(ctx)
		if err != nil {
			return err
		}
	case "pinboard":
		p := &pinboard.Pinboard{
			Token:        pinboardToken,
			Log:          cfg.Log,
			GraphQLToken: gqlToken,
		}
		err := p.UpdatePins(ctx)
		if err != nil {
			return err
		}
	case "random-tweets":
		t := &tweets.Twitter{
			TwitterAuth:  twitterAuth,
			Log:          cfg.Log,
			GraphQLToken: gqlToken,
		}
		err := t.CacheRandomTweets(ctx)
		if err != nil {
			return err
		}
	case "goodreads":
		g := &goodreads.Goodreads{
			Log:          cfg.Log,
			Token:        goodreadsToken,
			GraphQLToken: gqlToken,
		}
		err := g.UpsertBooks(ctx)
		if err != nil {
			return err
		}
	case "uptime":
		c := &uptime.Config{
			Log:       cfg.Log,
			ProjectID: GCPProject,
		}

		if err := uptime.UpdateUptimeChecks(ctx, c); err != nil {
			return err
		}

		if err := uptime.UpdateServices(ctx, c); err != nil {
			return err
		}
	case "stats":
		c := &stats.Config{
			Log:          cfg.Log,
			GraphQLToken: gqlToken,
			OWMKey:       os.Getenv("OPEN_WEATHER_MAP_KEY"),
		}

		if err := c.UpdateOften(ctx); err != nil {
			return err
		}
	case "code":
		c := &code.Config{
			Log:         cfg.Log,
			User:        "icco",
			GithubToken: githubToken,
			Cache:       cfg.Cache,
		}

		if err := c.FetchAndSaveCommits(ctx); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown job type: %q", job)
	}

	return nil
}
