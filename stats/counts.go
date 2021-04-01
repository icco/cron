package stats

import (
	"context"
	"fmt"

	gql "github.com/icco/graphql"
	"github.com/machinebox/graphql"
)

type countsResponse struct {
	Data struct {
		Counts []struct {
			Key   string  `json:"key"`
			Value float64 `json:"value"`
		} `json:"counts"`
	} `json:"data"`
	Error error `json:"error"`
}

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
	if resp.Error != nil {
		return nil, resp.Error
	}

	if len(resp.Data.Counts) == 0 {
		return nil, fmt.Errorf("count body was empty")
	}

	var stats []*gql.Stat
	for _, p := range resp.Data.Counts {
		stats = append(stats, &gql.Stat{Key: p.Key, Value: p.Value})
	}

	return stats, nil
}
