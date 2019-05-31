package spider

import (
	"context"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/jackdanger/collectlinks"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
)

type Config struct {
	Log *logrus.Logger
	URL string
}

var (
	c       *Config
	ops     uint64
	visited map[string]bool
)

func Crawl(octx context.Context, conf *Config) {
	c = conf

	queue := make(chan string, 100)
	visited = make(map[string]bool)
	ctx, cncl := context.WithTimeout(octx, 30*time.Second)

	go func() { queue <- c.URL }()

	for uri := range queue {
		enqueue(ctx, uri, queue)

		if ctx.Err() != nil {
			c.Log.Warn(ctx.Err())
			cncl()
			return
		}
	}

	cncl()
}

func enqueue(ctx context.Context, uri string, queue chan string) {
	atomic.AddUint64(&ops, 1)
	c.Log.WithContext(ctx).Printf("ops: %d, %s", atomic.LoadUint64(&ops), uri)

	if err := view.Register(
		ochttp.ClientSentBytesDistribution,
		ochttp.ClientReceivedBytesDistribution,
		ochttp.ClientRoundtripLatencyDistribution,
	); err != nil {
		c.Log.Fatal(err)
	}

	client := &http.Client{
		Transport: &ochttp.Transport{},
	}

	resp, err := client.Get(uri)
	visited[uri] = true
	if err != nil {
		c.Log.WithContext(ctx).WithError(err).Info("error scrapping")
		return
	}
	defer resp.Body.Close()

	links := collectlinks.All(resp.Body)

	for _, link := range links {
		absolute := fixUrl(link, uri)
		if uri != "" {
			if !visited[absolute] {
				go func() { queue <- absolute }()
			}
		}
	}
}

func fixUrl(href, base string) string {
	uri, err := url.Parse(href)
	if err != nil {
		return ""
	}
	baseUrl, err := url.Parse(base)
	if err != nil {
		return ""
	}
	uri = baseUrl.ResolveReference(uri)
	return uri.String()
}
