package spider

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type Config struct {
	Log *logrus.Logger
	URL string
}

var (
	c   *Config
	ops uint64
)

func Crawl(conf *Config) {
	c = conf

	messages := make(chan string, 100)
	results := make(chan string, 100)

	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)

	// This starts up 3 workers, initially blocked
	// because there are no jobs yet.
	for w := 0; w < 5; w++ {
		go worker(ctx, messages, results)
	}

	// Pass in init url
	messages <- c.URL
	for u := range results {
		messages <- u
	}
}

func worker(ctx context.Context, msgChan <-chan string, results chan<- string) {
	for msg := range msgChan {
		urls, err := ScrapeUrl(msg)

		atomic.AddUint64(&ops, 1)
		c.Log.WithContext(ctx).Printf("ops: %d, %s", atomic.LoadUint64(&ops), msg)

		if err != nil {
			c.Log.WithError(err).WithContext(ctx).Error("scrape error")
		}

		for _, u := range urls {
			results <- u
		}

		if ctx.Err() != nil {
			c.Log.Warn(ctx.Err())
			return
		}
	}
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
