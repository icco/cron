package cron

import (
	"context"
	"fmt"
	"os"

	"github.com/dgraph-io/ristretto"
	"github.com/icco/cron/code"
	"github.com/icco/cron/gaudit"
	"github.com/icco/cron/goodreads"
	"github.com/icco/cron/pinboard"
	"github.com/icco/cron/shared"
	"github.com/icco/cron/spider"
	"github.com/icco/cron/stats"
	"github.com/icco/cron/tweets"
	"github.com/icco/cron/updater"
)

const (
	// GCPProject is the project this runs in.
	GCPProject = "icco-cloud"

	// Service is the name of this service.
	Service = "cron"
)

// Config is our base act config struct.
type Config struct {
	shared.Config

	Cache *ristretto.Cache
}

// Act takes a job and calls a sub project to do work.
func (cfg *Config) Act(ctx context.Context, job string) error {
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
		if err != nil {
			return err
		}
		cfg.Log.Warnf("%d, %+v", v, err)
	case "minute":
		cfg.Log.Info("heartbeat")
	case "update-deployments":
		cfg := &updater.Config{
			Config:        shared.Config{Log: cfg.Log},
			GoogleProject: GCPProject,
		}
		if err := cfg.UpdateTriggers(ctx); err != nil {
			return fmt.Errorf("update triggers: %w", err)
		}
	case "github-audit":
		c := &gaudit.Config{
			Config:      shared.Config{Log: cfg.Log},
			User:        "icco",
			GithubToken: githubToken,
		}

		if err := c.CheckRepos(ctx); err != nil {
			return err
		}
	case "spider":
		spider.Crawl(ctx, &spider.Config{
			Config: shared.Config{Log: cfg.Log},
			URL:    "https://writing.natwelch.com/",
		})
	case "user-tweets":
		t := tweets.Twitter{
			Config:       shared.Config{Log: cfg.Log},
			TwitterAuth:  twitterAuth,
			GraphQLToken: gqlToken,
		}

		if err := t.CacophonyCron(ctx); err != nil {
			return err
		}

		if err := t.SaveUserTweets(ctx); err != nil {
			return err
		}
	case "pinboard":
		p := &pinboard.Pinboard{
			Config:       shared.Config{Log: cfg.Log},
			Token:        pinboardToken,
			GraphQLToken: gqlToken,
		}
		err := p.UpdatePins(ctx)
		if err != nil {
			return err
		}
	case "random-tweets":
		t := &tweets.Twitter{
			Config:       shared.Config{Log: cfg.Log},
			TwitterAuth:  twitterAuth,
			GraphQLToken: gqlToken,
		}
		err := t.CacheRandomTweets(ctx)
		if err != nil {
			return err
		}
	case "goodreads":
		g := &goodreads.Goodreads{
			Config:       shared.Config{Log: cfg.Log},
			Token:        goodreadsToken,
			GraphQLToken: gqlToken,
		}
		err := g.UpsertBooks(ctx)
		if err != nil {
			return err
		}
	case "stats":
		c := &stats.Config{
			Config:       shared.Config{Log: cfg.Log},
			GraphQLToken: gqlToken,
			OWMKey:       os.Getenv("OPEN_WEATHER_MAP_KEY"),
		}

		if err := c.UpdateOften(ctx); err != nil {
			return err
		}
	case "code":
		c := &code.Config{
			Config:      shared.Config{Log: cfg.Log},
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
