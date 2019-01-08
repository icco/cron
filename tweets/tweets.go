package tweets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/icco/cacophony/models"
)

func getUserTweets(ctx context.Context, consumerKey, consumerSecret, accessToken, accessSecret string) error {

	if consumerKey == "" || consumerSecret == "" || accessToken == "" || accessSecret == "" {
		return fmt.Errorf("Consumer key/secret and Access token/secret required")
	}

	config := oauth1.NewConfig(*consumerKey, *consumerSecret)
	token := oauth1.NewToken(*accessToken, *accessSecret)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	// Home Timeline
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
		}
		tm := time.Unix(i, 0)
		rtlimit := fmt.Errorf("Out of Rate Limit. Returns: %+v", tm)
		http.Error(w, rtlimit.Error(), http.StatusInternalServerError)
		return
	}

	if err != nil {
		log.WithError(err).Errorf("Error getting tweets: %+v", resp)
		return err
	}

	for _, t := range tweets {
		for _, u := range t.Entities.Urls {
			err = models.SaveURL(u.ExpandedURL, t.IDStr)
			if err != nil {
				log.WithError(err).Error("Error saving url")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	_, err = models.AllSavedURLs()
	if err != nil {
		log.WithError(err).Error("Error getting urls")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	w.Write([]byte(`"ok."`))
}
