package stats

import (
	"context"
	"fmt"

	gql "github.com/icco/graphql"
	"github.com/machinebox/graphql"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Config stores config.
type Config struct {
	Log          *logrus.Logger
	GraphQLToken string
	OWMKey       string
}

type keyFunc func(context.Context, *Config) (float64, error)

// funcMap is a list of stats to get. Some ideas:
// - Steps
// - Planes above
// - Devices on network
// - Blog posts
// - Books read this year
// - Tweets today
// - ETH price
// - Time coding
var funcMap = map[string]keyFunc{
	"ETH":                 GetETHPrice,
	"BTC":                 GetBTCPrice,
	"Aircraft Overhead":   GetAirplanes,
	"Beacon Temperature":  GetCurrentWeather("Beacon, NY, US"),
	"Chester Temperature": GetCurrentWeather("Chester, CA, US"),
	"London Temperature": GetCurrentWeather("London, UK"),
	"Santa Rosa Temperature": GetCurrentWeather("Santa Rosa, CA, US"),
	"Seattle Temperature": GetCurrentWeather("Seattle, WA, US"),
}

// UpdateOften updates stats that can be fetched quickly.
func (c *Config) UpdateOften(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for k, f := range funcMap {
		// https://golang.org/doc/faq#closures_and_goroutines
		k, f := k, f
		g.Go(func() error {
			v, err := f(ctx, c)
			if err != nil {
				return fmt.Errorf("get %q: %w", k, err)
			}

			return c.UploadStat(ctx, k, v)
		})
	}

	return g.Wait()
}

// UpdateRarely updates stats that should be fetched less frequently.
func (c *Config) UpdateRarely(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		stats, err := GetCounts(ctx, c)
		if err != nil {
			return fmt.Errorf("get counts: %w", err)
		}

		for _, s := range stats {
			if err := c.UploadStat(ctx, s.Key, s.Value); err != nil {
				return fmt.Errorf("upload stat: %w", err)
			}
		}

		return nil
	})

	return g.Wait()
}

// UploadStat uploads a single stat.
func (c *Config) UploadStat(ctx context.Context, key string, value float64) error {
	s := gql.NewStat{
		Key:   key,
		Value: value,
	}

	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	mut := `
  mutation ($s: NewStat!) {
      upsertStat(input: $s) {
        when
      }
    }
  `

	req := graphql.NewRequest(mut)
	req.Var("s", s)
	req.Header.Add("X-API-AUTH", c.GraphQLToken)
	req.Header.Add("User-Agent", "icco-cron/1.0")

	c.Log.WithField("stat", s).Debug("uploading stat")
	if err := gqlClient.Run(ctx, req, nil); err != nil {
		return fmt.Errorf("graphql: %w", err)
	}

	return nil
}
