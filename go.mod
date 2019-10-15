module github.com/icco/cron

go 1.13

require (
	cloud.google.com/go v0.46.3
	cloud.google.com/go/pubsub v1.0.1
	contrib.go.opencensus.io/exporter/stackdriver v0.12.7
	github.com/KyleBanks/goodreads v0.0.0-20190920105709-43e059021e8e
	github.com/dghubble/go-twitter v0.0.0-20190719072343-39e5462e111f
	github.com/dghubble/oauth1 v0.6.0
	github.com/felixge/httpsnoop v1.0.1
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/golang/protobuf v1.3.2
	github.com/google/go-github/v26 v26.1.3
	github.com/google/go-github/v28 v28.1.1
	github.com/hellofresh/logging-go v0.3.0
	github.com/icco/graphql v0.0.0-20190922160532-1e39d31dab34
	github.com/icco/logrus-stackdriver-formatter v0.3.0
	github.com/jackdanger/collectlinks v0.0.0-20160421202702-24c4ee2870ba
	github.com/machinebox/graphql v0.2.2
	github.com/pkg/errors v0.8.1
	github.com/sirupsen/logrus v1.4.2
	github.com/zachlatta/pin v0.0.0-20161031192518-51cb10fdcd53
	go.opencensus.io v0.22.1
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	google.golang.org/api v0.10.0
	google.golang.org/genproto v0.0.0-20191002211648-c459b9ce5143
	k8s.io/api v0.0.0-20190927115716-5d581ce610b0
	k8s.io/apimachinery v0.0.0-20191001195453-082230a5ffdd
	k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77
)
