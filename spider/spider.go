package spider

import (
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
	c        *Config
	messages chan string
)

func Crawl(conf *Config) {
	c = conf

	// Create channels for message passing.
	messages = make(chan string)

	// Pass in init url
	messages <- c.URL

	// Each channel will receive a value after some amount
	// of time, to simulate e.g. blocking RPC operations
	// executing in concurrent goroutines.
	go func() {
		select {
		case msg := <-messages:
			err := ScrapeUrl(msg)
			if err != nil {
				c.Log.WithError(err).Error("scrape error")
			}
		default:
			c.Log.Debug("no message received")
		}
	}()
	close(messages)
}

func ScrapeUrl(uri string) error {
	c.Log.Infof("visiting %+v", uri)
	response, err := http.Get(uri)

	if err != nil {
		return err
	}

	defer response.Body.Close()
	z := html.NewTokenizer(response.Body)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return nil
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
								messages <- attr.Val
							}
						}
					}
				}
			}
		}
	}
}
