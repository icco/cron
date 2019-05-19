package spider

import (
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/html"
)

func Crawl(url string) {
	// Create channels for message passing.
	messages := make(chan string)

	// Pass in init url
	messages <- url

	// Each channel will receive a value after some amount
	// of time, to simulate e.g. blocking RPC operations
	// executing in concurrent goroutines.
	go func() {
		select {
		case msg := <-messages:
			log.Println("received message", msg)
			ScrapeUrl(msg)
		default:
			log.Println("no message received")
		}
	}()
	close(messages)
}

func ScrapeUrl(uri string) ([]string, error) {
	response, err := http.Get(uri)
	ret := []string{}

	if err != nil {
		return nil, err
	} else {
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
									log.Printf("Found %+v", attr.Val)
									ret = append(ret, attr.Val)
								}
							}
						}
					}
				}
			}
		}
	}

	return ret, nil
}
