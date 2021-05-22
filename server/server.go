package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"cloud.google.com/go/pubsub"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/dgraph-io/ristretto"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/icco/cron"
	"github.com/icco/cron/sites"
	"github.com/icco/gutil/logging"
	"go.uber.org/zap"
)

var (
	log = logging.Must(logging.NewLogger(cron.Service))

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

	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Num keys to track frequency of (10M).
		MaxCost:     1 << 30, // Maximum cost of cache (1GB).
		BufferItems: 64,      // Number of keys per Get buffer.
	})
	if err != nil {
		log.Fatalw("could not create cache", zap.Error(err))
	}
	cfg := &cron.Config{Log: log, Cache: cache}

	if os.Getenv("USE_HTTP") == "" {
		go func() {
			ctx := context.Background()
			for {
				if err := recieveMessages(ctx, "cron-client", cfg); err != nil {
					log.Errorw("could not process message", zap.Error(err))
				}
			}
		}()
	}

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(logging.Middleware(log.Desugar(), cron.GCPProject))

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

	r.Post("/sub", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var event cloudevents.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			log.Errorw("could not decode request", zap.Error(err))
			http.Error(w, "body decode error", http.StatusInternalServerError)
			return
		}

		data := map[string]string{}
		if err := json.Unmarshal(event.DataEncoded, &data); err != nil {
			log.Warnw("could not decode json", zap.Error(err), "parsed", data, "unparsed", string(event.DataEncoded))
			http.Error(w, "body decode error", http.StatusInternalServerError)
			return
		}

		log.Debugw("got message", "parsed", data, "unparsed", string(event.DataEncoded))
		if err := cfg.Act(ctx, data["job"]); err != nil {
			log.Errorw("problem running job", "job", data, zap.Error(err))
			http.Error(w, "problem running job", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "success")
	})

	log.Fatal(http.ListenAndServe(":"+port, r))
}

func recieveMessages(ctx context.Context, subName string, cfg *cron.Config) error {
	pubsubClient, err := pubsub.NewClient(ctx, cron.GCPProject)
	if err != nil {
		return fmt.Errorf("create pubsub client: %w", err)
	}

	sub := pubsubClient.Subscription(subName)
	ok, err := sub.Exists(ctx)
	if err != nil {
		return fmt.Errorf("could not check exist of sub: %w", err)
	}
	if !ok {
		if _, err := pubsubClient.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{
			Topic: pubsubClient.Topic("cron"),
		}); err != nil {
			return fmt.Errorf("could not create sub: %w", err)
		}
	}

	scfg, err := sub.Config(ctx)
	if err != nil {
		return fmt.Errorf("could not get sub.Config: %w", err)
	}
	log.Debugw("got subscription config", "config", scfg, "subscription", subName)

	if err := sub.Receive(ctx, dealWithMessage(cfg)); err != nil && err != context.Canceled {
		return fmt.Errorf("recieving messages: %w", err)
	}

	return nil
}

func dealWithMessage(cfg *cron.Config) func(ctx context.Context, msg *pubsub.Message) {
	return func(ctx context.Context, msg *pubsub.Message) {
		cfg.Log.Debugw("got message", "msg", msg)
		data := map[string]string{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Warnw("could not decode json", zap.Error(err), "parsed", data, "unparsed", string(msg.Data))
			return
		}

		log.Debugw("got message", "parsed", data, "unparsed", string(msg.Data))
		if err := cfg.Act(ctx, data["job"]); err != nil {
			log.Errorw("problem running job", "job", data, zap.Error(err))
		}
		msg.Ack()
	}
}
