package main

import (
	"context"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/icco/cron"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

var (
	log = cron.InitLogging()
)

func main() {
	port := "8080"
	if fromEnv := os.Getenv("PORT"); fromEnv != "" {
		port = fromEnv
	}
	log.Printf("Starting up on http://localhost:%s", port)

	if os.Getenv("ENABLE_STACKDRIVER") != "" {
		labels := &stackdriver.Labels{}
		labels.Set("app", "cron", "The name of the current app.")
		sd, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:               "icco-cloud",
			MonitoredResource:       monitoredresource.Autodetect(),
			DefaultMonitoringLabels: labels,
			DefaultTraceAttributes:  map[string]interface{}{"app": "cron"},
		})

		if err != nil {
			log.Fatalf("Failed to create the Stackdriver exporter: %v", err)
		}
		defer sd.Flush()

		view.RegisterExporter(sd)
		trace.RegisterExporter(sd)
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	ctx := context.Background()
	pubsubClient, err := pubsub.NewClient(ctx, "icco-cloud")
	if err != nil {
		log.WithError(err).Fatal("Could not create client.")
	}

	sub, err := pubsubClient.CreateSubscription(ctx, "cron-client",
		pubsub.SubscriptionConfig{Topic: pubsubClient.Topic("cron")})
	if err != nil {
		log.WithError(err).Fatal("Could not create subscription.")
	}

	err = sub.Receive(context.Background(), func(ctx context.Context, m *pubsub.Message) {
		log.Printf("Got message: %s", m.Data)
		m.Ack()
	})
	if err != nil {
		log.WithError(err).Fatal("Could not recieve message.")
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cron.LoggingMiddleware())
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok."))
	})
	h := &ochttp.Handler{
		Handler:     r,
		Propagation: &propagation.HTTPFormat{},
	}
	if err := view.Register(ochttp.DefaultServerViews...); err != nil {
		log.WithError(err).Fatal("Failed to register ochttp.DefaultServerViews")
	}

	log.Fatal(http.ListenAndServe(":"+port, h))
}
