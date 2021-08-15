package pinboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/icco/cron/shared"
	"github.com/machinebox/graphql"
	"github.com/zachlatta/pin"
	"go.uber.org/zap"
)

// Pinboard contains the context needed to call pinboard.
type Pinboard struct {
	shared.Config

	Token        string
	GraphQLToken string
}

// UpdatePins gets and uploads pinned websites to graphql.
func (p *Pinboard) UpdatePins(ctx context.Context) error {
	tokenParts := strings.Split(p.Token, ":")
	if len(tokenParts) != 2 {
		return fmt.Errorf("Pinboard Token is malformed")
	}
	pinClient := pin.NewClient(nil, &pin.AuthToken{Username: tokenParts[0], Token: tokenParts[1]})

	tags := []string{}
	start := 0   // 0 means most recent
	results := 0 // 0 means all

	// Only get pins from the last 30m
	thirtyMin, err := time.ParseDuration("-30m")
	if err != nil {
		p.Log.Errorw("time parsing", zap.Error(err))
		return err
	}
	from := time.Now().Add(thirtyMin)
	to := time.Now()

	posts, _, err := pinClient.Posts.All(tags, start, results, &from, &to)
	if err != nil {
		p.Log.Errorw("failure talking to pinboard", zap.Error(err))
		return err
	}

	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	mut := `
    mutation ($title: String!, $tags: [String!]!, $uri: URI!, $desc: String!, $time: Time!) {
      upsertLink(
        input: {
          title: $title
          tags: $tags
          uri: $uri
          description: $desc
          created: $time
        }
      ) {
        id
      }
    }
  `
	for _, po := range posts {
		req := graphql.NewRequest(mut)
		req.Var("title", po.Title)
		req.Var("tags", po.Tags)
		req.Var("uri", po.URL)
		req.Var("desc", po.Description)
		req.Var("time", po.Time)
		req.Header.Add("X-API-AUTH", p.GraphQLToken)

		var resp json.RawMessage
		if err := gqlClient.Run(ctx, req, &resp); err != nil {
			p.Log.Errorw("graphql error on link upsert", zap.Error(err), "request", req)
			return err
		}
	}

	return nil
}
