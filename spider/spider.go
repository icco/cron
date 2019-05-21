package spider

import (
	"context"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type Config struct {
	Log *logrus.Logger
	URL string
}

var (
	c *Config
)

func Crawl(conf *Config) {
	c = conf

	// Create channels for message passing.
	messages := make(chan string, 100)

	// Pass in init url
	messages <- c.URL
	ctx := context.TODO()

	go worker(ctx, messages)
}

func worker(ctx context.Context, msgChan <-chan string) {
	c.Log.Debug("work")
	for {
		select {
		case msg := <-msgChan:
			urls, err := ScrapeUrl(msg)
			if err != nil {
				c.Log.WithError(err).WithContext(ctx).Error("scrape error")
			}

			for _, u := range urls {
				//msgChan <- u
				c.Log.Debug(u)
			}
		case <-ctx.Done():
			c.Log.Warn(ctx.Err())
			return
		}
	}
}

func ScrapeUrl(uri string) ([]string, error) {
	c.Log.Infof("visiting %+v", uri)
	response, err := http.Get(uri)
	ret := []string{}

	if err != nil {
		return ret, err
	}

	defer response.Body.Close()
	z := html.NewTokenizer(response.Body)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return ret, nil
		case tt == html.StartTagToken:
			t := z.Token()

			if t.Data == "a" {
				for _, attr := range t.Attr {
					if attr.Key == "href" {
						u, err := url.ParseRequestURI(attr.Val)
						if err != nil {
							continue
						} else {
							if u.IsAbs() {
								c.Log.Debugf("found %+v", attr.Val)
								ret = append(ret, attr.Val)
							}
						}
					}
				}
			}
		}
	}
}
