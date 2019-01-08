package main

import (
	"context"
	"os"

	"github.com/icco/cron/tweets"
	stackdriver "github.com/icco/logrus-stackdriver-formatter"
)

func main() {
	log := stackdriver.InitLogging()
	gqlToken := os.Getenv("GQL_TOKEN")
	err := tweets.SaveUserTweets(context.Background(), log, gqlToken, os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"), os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_SECRET"))
	log.WithError(err).Info("Finished")
}
