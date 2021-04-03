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
	"github.com/icco/gutil/logging"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/zap"
)

var (
	log = logging.Must(logging.NewLogger(cron.Service))

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
	log.Infow("Starting up", "host", fmt.Sprintf("http://localhost:%s", port))

	if os.Getenv("ENABLE_STACKDRIVER") != "" {
		labels := &stackdriver.Labels{}
		labels.Set("app", cron.Service, "The name of the current app.")
		sd, err := stackdriver.NewExporter(stackdriver.Options{
			ProjectID:               cron.GCPProject,
			MonitoredResource:       monitoredresource.Autodetect(),
			DefaultMonitoringLabels: labels,
			DefaultTraceAttributes:  map[string]interface{}{"app": cron.Service},
		})

		if err != nil {
			log.Fatalw("failed to create the stackdriver exporter", zap.Error(err))
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
				log.Fatalw("could not process message", zap.Error(err))
			}
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// r.Use(logging.Middleware(log.Desugar(), cron.GCPProject))
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok."))
		if err != nil {
			log.Errorw("could not write response", zap.Error(err))
		}
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.New("root").Parse(rootTmpl)
		if err != nil {
			log.Errorw("could not parse template", zap.Error(err))
		}

		data := []string{
			fmt.Sprintf("%d sites", len(sites.All)),
		}
		if err := tmpl.Execute(w, data); err != nil {
			log.Errorw("could not write response", zap.Error(err))
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
		log.Fatalw("failed to register ochttp views", zap.Error(err))
	}

	if err := view.Register([]*view.View{
		msgRecvView,
		msgAckView,
	}...); err != nil {
		log.Fatalw("failed to register metrics", zap.Error(err))
	}

	log.Fatal(http.ListenAndServe(":"+port, h))
}

func recieveMessages(ctx context.Context, subName string) error {
	pubsubClient, err := pubsub.NewClient(ctx, "icco-cloud")
	if err != nil {
		log.Errorw("cloudn't create client", zap.Error(err))
		return err
	}

	sub, err := pubsubClient.CreateSubscription(ctx, subName,
		pubsub.SubscriptionConfig{Topic: pubsubClient.Topic("cron")})
	if err != nil {
		// This is fine, don't do anything.
		sub = pubsubClient.Subscription(subName)
	}

	err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		stats.Record(ctx, msgRecv.M(1))

		data := map[string]string{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Warnw("could not decode json", zap.Error(err), "parsed", data, "unparsed", string(msg.Data))
		} else {
			log.Debugw("got message", "parsed", data, "unparsed", string(msg.Data))
			if err := cron.Act(ctx, data["job"]); err != nil {
				log.Errorw("problem running job", "job", data, zap.Error(err))
			}
			msg.Ack()

			stats.Record(ctx, msgAck.M(1))
		}
	})

	if err != nil {
		return fmt.Errorf("recieving messages: %w", err)
	}

	return nil
}
