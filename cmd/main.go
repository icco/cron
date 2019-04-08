package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/icco/cron"
)

var (
	log = cron.InitLogging()
)

func main() {
	cmd := os.Args[1:]
	if len(cmd) < 2 || cmd[0] != "send" {
		fmt.Printf("Usage: $ %s send message", os.Args[0])
		return
	}

	err := cron.Act(context.Background(), strings.Join(cmd[1:], " "))
	if err != nil {
		log.WithError(err).Error(err.Error())
		return
	}
}
