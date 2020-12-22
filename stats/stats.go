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
const funcMap = map[string]keyFunc{
	"ETH": GetETHPrice,
}

// Update gets all stats.
func (c *Config) Update(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for k, f := range funcMap {
		// https://golang.org/doc/faq#closures_and_goroutines
		k, f := k, f
		g.Go(func() error {
			v, err := f(ctx)
			if err != nil {
				return err
			}

			return c.UploadStat(ctx, k, v)
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
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
      upsertStat(input: $b) {
        modified
      }
    }
  `

	req := graphql.NewRequest(mut)
	req.Var("s", s)
	req.Header.Add("X-API-AUTH", g.GraphQLToken)
	req.Header.Add("User-Agent", "icco-cron/1.0")

	if err := gqlClient.Run(ctx, req, nil); err != nil {
		return fmt.Errorf("graphql: %w", err)
	}

	return nil
}
