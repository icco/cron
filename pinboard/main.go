package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/machinebox/graphql"
	"github.com/zachlatta/pin"
)

func main() {
	tokenParts := strings.Split(os.Getenv("PINBOARD_TOKEN"), ":")
	pinClient := pin.NewClient(nil, &pin.AuthToken{Username: tokenParts[0], Token: tokenParts[1]})

	tags := []string{}
	start := 0   // 0 means most recent
	results := 0 // 0 means all

	// Only get pins from the last 90d
	oneDay, err := time.ParseDuration("-24h")
	if err != nil {
		log.Fatalf("time parsing: %+v", err)
	}
	from := time.Now().Add(oneDay * 90)
	to := time.Now()

	posts, _, err := pinClient.Posts.All(tags, start, results, &from, &to)
	if err != nil {
		log.Panicf("error talking to pinboard: %+v", err)
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
	gqlClient.Log = func(s string) { log.Println(s) }

	for _, p := range posts {
		req := graphql.NewRequest(mut)
		req.Var("title", p.Title)
		req.Var("tags", p.Tags)
		req.Var("uri", p.URL)
		req.Var("desc", p.Description)
		req.Var("time", p.Time)
		req.Header.Add("Authorization", os.Getenv("GQL_TOKEN"))

		err := gqlClient.Run(context.Background(), req, nil)
		if err != nil {
			log.Panicf("error talking to graphql: %+v", err)
		}
	}

	req := graphql.NewRequest(`query { counts { key, value } }`)
	var resp json.RawMessage
	err = gqlClient.Run(context.Background(), req, &resp)
	if err != nil {
		log.Panicf("error talking to graphql: %+v", err)
	}
	log.Printf("New Database Counts: %+v", string(resp))
}
