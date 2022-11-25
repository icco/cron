package spider

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/icco/cron/shared"
	"github.com/jackdanger/collectlinks"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"
)

// Config is our config.
type Config struct {
	shared.Config

	URL string
}

var (
	c       *Config
	ops     uint64
	visited map[string]bool
)

// Crawl begins a crawl.
func Crawl(octx context.Context, conf *Config) {
	c = conf

	queue := make(chan string, 100)
	visited = make(map[string]bool)
	ctx, cncl := context.WithTimeout(octx, 30*time.Second)

	go func() { queue <- c.URL }()

	for uri := range queue {
		enqueue(ctx, uri, queue)

		if ctx.Err() != nil {
			c.Log.Warnw("error crawling", zap.Error(ctx.Err()))
			cncl()
			return
		}
	}

	cncl()
}

func enqueue(ctx context.Context, uri string, queue chan string) {
	atomic.AddUint64(&ops, 1)
	c.Log.Infow("enqued", "ops", atomic.LoadUint64(&ops), "uri", uri)

	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	resp, err := client.Get(uri)
	visited[uri] = true
	if err != nil {
		c.Log.Infow("error scrapping", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	links := collectlinks.All(resp.Body)

	for _, link := range links {
		absolute := fixURL(link, uri)
		if uri != "" {
			if !visited[absolute] {
				go func() { queue <- absolute }()
			}
		}
	}
}

func fixURL(href, base string) string {
	uri, err := url.Parse(href)
	if err != nil {
		return ""
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}
	uri = baseURL.ResolveReference(uri)
	return uri.String()
}
