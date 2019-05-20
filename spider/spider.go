package spider

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

type Config struct {
	Log *logrus.Logger
	URL string
}

var (
	c  *Config
	wg sync.WaitGroup
)

func Crawl(conf *Config) {
	c = conf

	// Create channels for message passing.
	messages := make(chan string, 100)

	// Pass in init url
	messages <- c.URL
	wg.Add(1)

	go worker(messages)

	wg.Wait()
	defer close(messages)
}

func worker(msgChan chan string) {
	defer wg.Done()

	for msg := range msgChan {
		err := ScrapeUrl(msg, msgChan)
		if err != nil {
			c.Log.WithError(err).Error("scrape error")
		}
	}
}

func ScrapeUrl(uri string, msgChan chan string) error {
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
								msgChan <- attr.Val
							}
						}
					}
				}
			}
		}
	}
}
