package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/icco/cron"
	"github.com/icco/cron/sites"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

var (
	log = cron.InitLogging()

	msgRecv     = stats.Int64("natwelch.com/stats/message/received", "received message from Pub/Sub", stats.UnitDimensionless)
	msgRecvView = &view.View{
		Name:        "natwelch.com/views/message/received",
		Description: "received message from Pub/Sub",
		Measure:     msgRecv,
		Aggregation: view.Count(),
	}

	msgAck     = stats.Int64("natwelch.com/stats/message/acknowledged", "acknowledged message from Pub/Sub", stats.UnitDimensionless)
	msgAckView = &view.View{
		Name:        "natwelch.com/views/message/acknowledged",
		Description: "acknowledged message from Pub/Sub",
		Measure:     msgRecv,
		Aggregation: view.Count(),
	}

	rootTmpl = `
<html>
<head>
<title>Cron!</title>
</head>
<body>
<h1>Cron Party!</h1>
<ul>
{{ range . }}
<li>{{ . }}</li>
{{ end }}
</ul>
</body>
</html>
`
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
			log.WithError(err).Fatalf("failed to create the stackdriver exporter")
		}
		defer sd.Flush()

		view.RegisterExporter(sd)
		trace.RegisterExporter(sd)
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	go func() {
		ctx := context.Background()
		for {
			err := recieveMessages(ctx, "cron-client")
			if err != nil {
				log.WithError(err).Fatal("could not process message")
			}
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cron.LoggingMiddleware())
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok."))
		if err != nil {
			log.WithError(err).Error("could not write response")
		}
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("root").Parse(rootTmpl)
		if err != nil {
			log.WithError(err).Error("could not parse template")
		}

		data := []string{}
		if err := tmpl.Execute(w, data); err != nil {
			log.WithError(err).Error("could not write response")
		}
	})
	h := &ochttp.Handler{
		Handler:     r,
		Propagation: &propagation.HTTPFormat{},
	}
	if err := view.Register([]*view.View{
		ochttp.ServerRequestCountView,
		ochttp.ServerResponseCountByStatusCode,
	}...); err != nil {
		log.WithError(err).Fatal("Failed to register ochttp views")
	}

	if err := view.Register([]*view.View{
		msgRecvView,
		msgAckView,
	}...); err != nil {
		log.WithError(err).Fatal("Failed to register server metrics")
	}

	log.Fatal(http.ListenAndServe(":"+port, h))
}

func recieveMessages(ctx context.Context, subName string) error {
	pubsubClient, err := pubsub.NewClient(ctx, "icco-cloud")
	if err != nil {
		log.WithError(err).Fatal("Could not create client.")
		return err
	}

	sub, err := pubsubClient.CreateSubscription(ctx, subName,
		pubsub.SubscriptionConfig{Topic: pubsubClient.Topic("cron")})
	if err != nil {
		// This is fine, don't do anything.
		log.WithError(err).Info("Could not create subscription.")
		sub = pubsubClient.Subscription(subName)
	}

	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		stats.Record(ctx, msgRecv.M(1))

		data := map[string]string{
			fmt.Sprintf("%d sites", len(sites.All)),
		}
		err := json.Unmarshal(msg.Data, &data)
		logFields := logrus.Fields{"parsed": data, "unparsed": string(msg.Data)}

		if err != nil {
			log.WithError(err).WithFields(logFields).Warn("Couldn't decode json.")
		} else {
			log.WithFields(logFields).Debug("Got message")
			err = cron.Act(ctx, data["job"])
			if err != nil {
				log.WithError(err).Error("Problem running job.")
			}
			msg.Ack()

			stats.Record(ctx, msgAck.M(1))
		}
	})

	if err != nil {
		return errors.Wrap(err, "recieving messages")
	}

	return nil
}
