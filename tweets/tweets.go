package tweets

import (
	"context"
	//"encoding/json"
	"fmt"
	"math/rand"
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

type Twitter struct {
	TwitterAuth  *TwitterAuth
	Log          *logrus.Logger
	GraphQLToken string
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

func (t *Twitter) SaveUserTweets(ctx context.Context) error {
	client, user, err := t.TwitterAuth.Validate(ctx, t.Log)
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
			t.Log.WithError(err).Error("Error converting int")
			return err
		}
		tm := time.Unix(i, 0)
		return fmt.Errorf("Out of Rate Limit. Returns: %+v", tm)
	}

	if err != nil {
		t.Log.WithError(err).Errorf("Error getting tweets: %+v", resp)
		return err
	}

	for _, tw := range tweets {
		err := t.UploadTweet(ctx, tw)
		if err != nil {
			return nil
		}
	}

	return nil
}

type tweetids struct {
	HomeTimelineURLs []struct {
		TweetIDs []string `json:"tweetIDs"`
	} `json:"homeTimelineURLs"`
}

func (t *Twitter) CacheRandomTweets(ctx context.Context) error {
	query := `query {
    homeTimelineURLs {
      tweetIDs
    }
  }
  `

	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	//gqlClient.t.Log = func(s string) { t.Log.Debug(s) }

	var data tweetids

	req := graphql.NewRequest(query)
	req.Header.Add("X-API-AUTH", t.GraphQLToken)
	req.Header.Add("User-Agent", "icco-cron/1.0")
	err := gqlClient.Run(ctx, req, &data)
	if err != nil {
		t.Log.WithError(err).Error("error talking to graphql")
		return err
	}

	ids := []string{}
	for _, u := range data.HomeTimelineURLs {
		ids = append(ids, u.TweetIDs...)
	}

	for i := 0; i < 10; i++ {
		idString := ids[rand.Intn(len(ids))]
		id, err := strconv.ParseInt(idString, 10, 64)
		if err != nil {
			return err
		}

		tw, err := t.GetTweet(ctx, id)
		if err != nil {
			return err
		}

		if tw != nil {
			err = t.UploadTweet(ctx, *tw)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Twitter) GetTweet(ctx context.Context, id int64) (*twitter.Tweet, error) {
	client, _, err := t.TwitterAuth.Validate(ctx, t.Log)
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
			t.Log.WithError(err).Error("Error converting int")
			return nil, err
		}
		tm := time.Unix(i, 0)
		return nil, fmt.Errorf("Out of Rate Limit. Returns: %+v", tm)
	}

	if err != nil {
		t.Log.WithError(err).Errorf("Error getting tweets: %+v", resp)
		return nil, err
	}

	return tweet, nil
}

func (t *Twitter) UploadTweet(ctx context.Context, tw twitter.Tweet) error {
	text := tw.FullText
	if text == "" && tw.Text != "" {
		text = tw.Text
	}

	if tw.ExtendedTweet != nil && tw.ExtendedTweet.FullText != "" {
		text = tw.ExtendedTweet.FullText
	}

	tweet := gql.NewTweet{
		ID:            tw.IDStr,
		Text:          text,
		ScreenName:    tw.User.ScreenName,
		FavoriteCount: tw.FavoriteCount,
		RetweetCount:  tw.RetweetCount,
		Hashtags:      make([]string, len(tw.Entities.Hashtags)),
		Symbols:       []string{},
		UserMentions:  make([]string, len(tw.Entities.UserMentions)),
		Urls:          make([]gql.URI, len(tw.Entities.Urls)),
	}

	tp, err := tw.CreatedAtTime()
	if err != nil {
		return err
	}
	tweet.Posted = tp

	for i, v := range tw.Entities.Hashtags {
		tweet.Hashtags[i] = v.Text
	}

	for i, v := range tw.Entities.Urls {
		tweet.Urls[i] = gql.NewURI(v.ExpandedURL)
	}

	for i, v := range tw.Entities.UserMentions {
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

	req := graphql.NewRequest(mut)
	req.Var("t", tweet)
	req.Header.Add("X-API-AUTH", t.GraphQLToken)
	req.Header.Add("User-Agent", "icco-cron/1.0")
	err = gqlClient.Run(ctx, req, nil)
	if err != nil {
		t.Log.WithError(err).Error("error talking to graphql")
		return err
	}

	return nil
}
