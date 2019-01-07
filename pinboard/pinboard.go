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

func UpdatePins(ctx context.Context, log *logrus.Logger, pinboardToken, graphqlToken string) error {
	if pinboardToken == "" {
		return fmt.Errorf("Pinboard Token is empty")
	}

	if graphqlToken == "" {
		return fmt.Errorf("GraphQL Token is empty")
	}

	tokenParts := strings.Split(pinboardToken, ":")
	if len(tokenParts) != 2 {
		return fmt.Errorf("Pinboard Token is malformed")
	}
	pinClient := pin.NewClient(nil, &pin.AuthToken{Username: tokenParts[0], Token: tokenParts[1]})

	tags := []string{}
	start := 0   // 0 means most recent
	results := 0 // 0 means all

	// Only get pins from the last 90d
	oneDay, err := time.ParseDuration("-24h")
	if err != nil {
		log.WithError(err).Error("time parsing")
		return err
	}
	from := time.Now().Add(oneDay * 90)
	to := time.Now()

	posts, _, err := pinClient.Posts.All(tags, start, results, &from, &to)
	if err != nil {
		log.WithError(err).Error("failure talking to pinboard")
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
	gqlClient.Log = func(s string) { log.Debug(s) }

	for _, p := range posts {
		req := graphql.NewRequest(mut)
		req.Var("title", p.Title)
		req.Var("tags", p.Tags)
		req.Var("uri", p.URL)
		req.Var("desc", p.Description)
		req.Var("time", p.Time)
		req.Header.Add("Authorization", graphqlToken)

		err := gqlClient.Run(ctx, req, nil)
		if err != nil {
			log.WithError(err).Error("error talking to graphql")
			return err
		}
	}

	req := graphql.NewRequest(`query { counts { key, value } }`)
	var resp json.RawMessage
	err = gqlClient.Run(ctx, req, &resp)
	if err != nil {
		log.WithError(err).Error("error talking to graphql")
		return err
	}

	log.WithField("link_count", string(resp)).Infof("New Database Counts: %+v", string(resp))

	return nil
}
