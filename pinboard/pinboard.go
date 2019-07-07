package pinboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	"github.com/sirupsen/logrus"
	"github.com/zachlatta/pin"
)

// Pinboard contains the context needed to call pinboard.
type Pinboard struct {
	Token        string
	Log          *logrus.Logger
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
		p.Log.WithError(err).Error("time parsing")
		return err
	}
	from := time.Now().Add(thirtyMin)
	to := time.Now()

	posts, _, err := pinClient.Posts.All(tags, start, results, &from, &to)
	if err != nil {
		p.Log.WithError(err).Error("failure talking to pinboard")
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

		err := gqlClient.Run(ctx, req, nil)
		if err != nil {
			p.Log.WithError(err).Error("error talking to graphql")
			return err
		}
	}

	req := graphql.NewRequest(`query { counts { key, value } }`)
	var resp json.RawMessage
	err = gqlClient.Run(ctx, req, &resp)
	if err != nil {
		p.Log.WithError(err).Error("error talking to graphql")
		return err
	}

	p.Log.WithField("link_count", string(resp)).Infof("New Database Counts: %+v", string(resp))

	return nil
}
