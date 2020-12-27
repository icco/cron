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
	Error error `json:"error"`
}

func GetCounts(ctx context.Context, cfg *Config) ([]*gql.Stat, error) {
	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	req := graphql.NewRequest(`query { counts { key, value } }`)
	var resp countsResponse
	if err := gqlClient.Run(ctx, req, &resp); err != nil {
		return nil, err
	}

	cfg.Log.WithField("response", resp).Debug("got count response")
	if resp.Error != nil {
		return nil, resp.Error
	}

	if len(res.Data.Counts) == 0 {
		return nil, fmt.Errorf("count body was empty")
	}

	var stats []*gql.Stat
	for _, p := range resp.Data.Counts {
		stats = append(stats, &gql.Stat{Key: p.Key, Value: p.Value})
	}

	return stats, nil
}
