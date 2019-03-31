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

type TwitterAuth struct {
	ConsumerKey    string
	ConsumerSecret string
	AccessToken    string
	AccessSecret   string
}

func (t *TwitterAuth) Validate(ctx context.Context, log *logrus.Logger) (*twitter.Client, *twitter.User, error) {
	if t.ConsumerKey == "" || t.ConsumerSecret == "" || t.AccessToken == "" || t.AccessSecret == "" {
		return nil, nil, fmt.Errorf("Consumer key/secret and Access token/secret required")
	}

	config := oauth1.NewConfig(t.ConsumerKey, t.ConsumerSecret)
	token := oauth1.NewToken(t.AccessToken, t.AccessSecret)
	httpClient := config.Client(ctx, token)
	client := twitter.NewClient(httpClient)

	// Verify Credentials
	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(true),
	}
	user, resp, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		log.WithError(err).WithField("response", resp).Errorf("error verifying creds")
		return nil, nil, err
	}

	return client, user, nil
}

func SaveUserTweets(ctx context.Context, log *logrus.Logger, graphqlToken string, tAuth *TwitterAuth) error {
	if graphqlToken == "" {
		return fmt.Errorf("GraphQL Token is empty")
	}

	client, user, err := tAuth.Validate(ctx, log)
	if err != nil {
		return err
	}

	userTimelineParams := &twitter.UserTimelineParams{
		ScreenName:      user.ScreenName,
		Count:           200,
		TweetMode:       "extended",
		IncludeRetweets: twitter.Bool(true),
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

func CacheRandomTweets(ctx context.Context, log *logrus.Logger, graphqlToken string, tAuth *TwitterAuth) error {
	query := `query {
    homeTimelineURLs {
      tweetIDs
      tweets {
        id
      }
    }
  }
  `

	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	gqlClient.Log = func(s string) { log.Debug(s) }

	var resp interface{}
	req := graphql.NewRequest(query)
	req.Header.Add("X-API-AUTH", graphqlToken)
	err := gqlClient.Run(ctx, req, resp)
	if err != nil {
		log.WithError(err).Error("error talking to graphql")
		return err
	}

	log.Debugf("%+v", resp)

	return nil
}

func GetTweet(ctx context.Context, log *logrus.Logger, tAuth *TwitterAuth, id int64) (*twitter.Tweet, error) {
	client, _, err := tAuth.Validate(ctx, log)
	if err != nil {
		return nil, err
	}

	params := &twitter.StatusShowParams{
		IncludeEntities: twitter.Bool(true),
	}

	tweet, resp, err := client.Statuses.Show(id, params)
	if resp.Header.Get("X-Rate-Limit-Remaining") == "0" {
		i, err := strconv.ParseInt(resp.Header.Get("X-Rate-Limit-Reset"), 10, 64)
		if err != nil {
			log.WithError(err).Error("Error converting int")
			return nil, err
		}
		tm := time.Unix(i, 0)
		return nil, fmt.Errorf("Out of Rate Limit. Returns: %+v", tm)
	}

	if err != nil {
		log.WithError(err).Errorf("Error getting tweets: %+v", resp)
		return nil, err
	}

	return tweet, nil
}

func UploadTweet(ctx context.Context, log *logrus.Logger, graphqlToken string, t twitter.Tweet) error {

	// I have no idea if this is right.
	// https://developer.twitter.com/en/docs/tweets/data-dictionary/overview/tweet-object
	//log.WithField("tweet", t).Debug("examining text fields")
	text := t.FullText
	if text == "" && t.Text != "" {
		text = t.Text
	}

	if t.ExtendedTweet != nil && t.ExtendedTweet.FullText != "" {
		text = t.ExtendedTweet.FullText
	}

	// This is broken
	//	if t.Retweeted {
	//		if t.RetweetedStatus != nil {
	//			err := UploadTweet(ctx, log, graphqlToken, *t.RetweetedStatus)
	//			if err != nil {
	//				log.WithError(err).Error("Error posting retweet")
	//			}
	//		}
	//	}

	tweet := gql.NewTweet{
		ID:            t.IDStr,
		Text:          text,
		ScreenName:    t.User.ScreenName,
		FavoriteCount: t.FavoriteCount,
		RetweetCount:  t.RetweetCount,
		Hashtags:      make([]string, len(t.Entities.Hashtags)),
		Symbols:       []string{},
		UserMentions:  make([]string, len(t.Entities.UserMentions)),
		Urls:          make([]gql.URI, len(t.Entities.Urls)),
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
		tweet.Urls[i] = gql.NewURI(v.ExpandedURL)
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
	req.Header.Add("X-API-AUTH", graphqlToken)
	err = gqlClient.Run(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("error talking to graphql")
		return err
	}

	return nil
}
