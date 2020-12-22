package stats

import (
	"context"

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
}

func GetCounts(ctx context.Context, cfg *Config) ([]*gql.Stat, error) {
	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	req := graphql.NewRequest(`query { counts { key, value } }`)
	var resp countsResponse
	if err := gqlClient.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	var stats []*gql.Stat
	for _, p := range resp.Data.Counts {
		stats = append(stats, &gql.Stat{Key: p.Key, Value: p.Value})
	}

	return stats, nil
}
