package stats

import (
	"context"
	"fmt"

	gql "github.com/icco/graphql"
	"github.com/machinebox/graphql"
)

type countsResponse struct {
	Counts []*gql.Stat
}

// GetCounts gets counts from graphql.
func GetCounts(ctx context.Context, cfg *Config) ([]*gql.Stat, error) {
	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	gqlClient.Log = func(s string) { cfg.Log.Debug(s) }

	req := graphql.NewRequest(`query { counts { key, value } }`)
	req.Header.Add("X-API-AUTH", cfg.GraphQLToken)
	req.Header.Add("User-Agent", "icco-cron/1.0")
	var resp countsResponse
	if err := gqlClient.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	cfg.Log.Debugw("got count response", "response", resp)
	if len(resp.Counts) == 0 {
		return nil, fmt.Errorf("count body was empty")
	}

	return resp.Counts, nil
}
