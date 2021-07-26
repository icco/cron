package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/dgraph-io/ristretto"
	"github.com/icco/cron"
	"github.com/icco/cron/shared"
	"github.com/icco/gutil/logging"
	"go.uber.org/zap"
)

var (
	log = logging.Must(logging.NewLogger(cron.Service))
)

func main() {
	cmd := os.Args[1:]
	if len(cmd) < 2 || cmd[0] != "send" {
		fmt.Printf("Usage: $ %s send message", os.Args[0])
		return
	}

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Num keys to track frequency of (10M).
		MaxCost:     1 << 30, // Maximum cost of cache (1GB).
		BufferItems: 64,      // Number of keys per Get buffer.
	})
	if err != nil {
		log.Fatalw("could not create cache", zap.Error(err))
	}
	cfg := &cron.Config{
		Config: shared.Config{Log: log},
		Cache:  cache,
	}

	if err := cfg.Act(context.Background(), strings.Join(cmd[1:], " ")); err != nil {
		log.Errorw("could not act", zap.Error(err))
		return
	}
}
