package spider

import (
	"context"
	"net/http"
	"net/url"
	"time"

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

	messages := make(chan string, 100)

	// Pass in init url
	messages <- c.URL
	ctx, cncl := context.WithTimeout(context.Background(), 30*time.Second)

	worker(ctx, messages)
	cncl()
}

func worker(ctx context.Context, msgChan chan string) {
	for msg := range msgChan {
		urls, err := ScrapeUrl(msg)
		if err != nil {
			c.Log.WithError(err).WithContext(ctx).Error("scrape error")
		}

		for _, u := range urls {
			msgChan <- u
		}

		if ctx.Err() != nil {
			c.Log.Warn(ctx.Err())
			return
		}
	}

	return
}

func ScrapeUrl(uri string) ([]string, error) {
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
								ret = append(ret, attr.Val)
							}
						}
					}
				}
			}
		}
	}
}
