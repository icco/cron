package tweets

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	gql "github.com/icco/graphql"
	"github.com/machinebox/graphql"
	"github.com/sirupsen/logrus"
)

func SaveUserTweets(ctx context.Context, log *logrus.Logger, graphqlToken, consumerKey, consumerSecret, accessToken, accessSecret string) error {
	if graphqlToken == "" {
		return fmt.Errorf("GraphQL Token is empty")
	}

	if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessSecret == "" {
		return fmt.Errorf("Consumer key/secret and Access token/secret required")
	}

	config := oauth1.NewConfig(consumerKey, consumerSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	// Verify Credentials
	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	user, resp, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		log.WithError(err).Errorf("Error verifying creds: %+v", resp)
		return err
	}

	userTimelineParams := &twitter.UserTimelineParams{
		ScreenName: user.ScreenName,
		Count:      200,
		TweetMode:  "extended",
	}
	tweets, resp, err := client.Timelines.UserTimeline(userTimelineParams)
	if resp.Header.Get("X-Rate-Limit-Remaining") == "0" {
		i, err := strconv.ParseInt(resp.Header.Get("X-Rate-Limit-Reset"), 10, 64)
		if err != nil {
			log.WithError(err).Error("Error converting int")
			return err
		}
		tm := time.Unix(i, 0)
		return fmt.Errorf("Out of Rate Limit. Returns: %+v", tm)
	}

	if err != nil {
		log.WithError(err).Errorf("Error getting tweets: %+v", resp)
		return err
	}

	for _, t := range tweets {
		err := UploadTweet(ctx, log, graphqlToken, t)
		if err != nil {
			return nil
		}
	}

	return nil
}

func UploadTweet(ctx context.Context, log *logrus.Logger, graphqlToken string, t twitter.Tweet) error {
	tweet := gql.NewTweet{
		ID:            t.IDStr,
		Text:          t.FullText,
		ScreenName:    t.User.ScreenName,
		FavoriteCount: &t.FavoriteCount,
		RetweetCount:  &t.RetweetCount,
		Hashtags:      make([]string, len(t.Entities.Hashtags)),
		Symbols:       []string{},
		UserMentions:  make([]string, len(t.Entities.UserMentions)),
		Urls:          make([]string, len(t.Entities.Urls)),
	}

	tp, err := t.CreatedAtTime()
	if err != nil {
		return err
	}
	tweet.Posted = tp

	for i, v := range t.Entities.Hashtags {
		tweet.Hashtags[i] = v.Text
	}

	for i, v := range t.Entities.Urls {
		tweet.Urls[i] = v.ExpandedURL
	}

	for i, v := range t.Entities.UserMentions {
		tweet.UserMentions[i] = v.ScreenName
	}

	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	mut := `
  mutation ($t: NewTweet!) {
      upsertTweet(input: $t) {
        id
      }
    }
  `
	gqlClient.Log = func(s string) { log.Debug(s) }

	req := graphql.NewRequest(mut)
	req.Var("t", tweet)
	req.Header.Add("Authorization", graphqlToken)
	err = gqlClient.Run(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("error talking to graphql")
		return err
	}

	return nil
}
