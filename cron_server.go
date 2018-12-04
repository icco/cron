package main

import (
	"context"
	"os"

	"cloud.google.com/go/pubsub"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	log = &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.JSONFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
)

func main() {
	if os.Getenv("ENABLE_STACKDRIVER") != "" {
		sd, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:               "icco-cloud",
			MetricPrefix:            "cron",
			MonitoredResource:       monitoredresource.Autodetect(),
			DefaultMonitoringLabels: &stackdriver.Labels{},
		})

		if err != nil {
			log.Fatalf("Failed to create the Stackdriver exporter: %v", err)
		}
		defer sd.Flush()

		trace.RegisterExporter(sd)
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	ctx := context.Background()
	pubsubClient, err := pubsub.NewClient(ctx, "icco-cloud")
	if err != nil {
		// TODO: Handle error.
	}

	sub, err := pubsubClient.CreateSubscription(ctx, "cron-client",
		pubsub.SubscriptionConfig{Topic: pubsubClient.Topic("cron")})
	if err != nil {
		// TODO: Handle error.
	}

	err = sub.Receive(context.Background(), func(ctx context.Context, m *pubsub.Message) {
		log.Printf("Got message: %s", m.Data)
		m.Ack()
	})
	if err != nil {
		// Handle error.
	}
}
