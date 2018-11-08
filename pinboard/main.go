package main

import (
	"log"
	"os"
	"strings"

	"github.com/zachlatta/pin"
	//"github.com/machinebox/graphql"
)

func main() {

	tokenParts := strings.Split(os.Getenv("PINBOARD_TOKEN"), ":")
	pinClient := pin.NewClient(nil, &pin.AuthToken{Username: tokenParts[0], Token: tokenParts[1]})

	tags := []string{}
	start := 0    // 0 means beginning
	results := 10 // 0 means all

	posts, _, err := pinClient.Posts.All(tags, start, results, nil, nil)
	if err != nil {
		log.Panicf("error talking to pinboard: %+v", err)
	}

	for _, p := range posts {
		log.Printf("%+v", p)
	}
}
