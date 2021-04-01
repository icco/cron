package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/icco/cron"
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

	if err := cron.Act(context.Background(), strings.Join(cmd[1:], " ")); err != nil {
		log.Errorw("could not act", zap.Error(err))
		return
	}
}
