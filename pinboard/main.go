package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/machinebox/graphql"
	"github.com/zachlatta/pin"
)

func main() {
	tokenParts := strings.Split(os.Getenv("PINBOARD_TOKEN"), ":")
	pinClient := pin.NewClient(nil, &pin.AuthToken{Username: tokenParts[0], Token: tokenParts[1]})

	tags := []string{}
	start := 0   // 0 means beginning
	results := 0 // 0 means all

	posts, _, err := pinClient.Posts.All(tags, start, results, nil, nil)
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

		var resp json.RawMessage
		err := gqlClient.Run(context.Background(), req, &resp)
		if err != nil {
			log.Panicf("error talking to graphql: %+v", err)
		}
	}
}
