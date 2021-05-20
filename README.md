# cron

[![GoDoc](https://godoc.org/github.com/icco/cron?status.svg)](https://godoc.org/github.com/icco/cron)
[![Go Report Card](https://goreportcard.com/badge/github.com/icco/cron)](https://goreportcard.com/report/github.com/icco/cron)
[![Build Status](https://travis-ci.com/icco/cron.svg?branch=main)](https://travis-ci.com/icco/cron)

A bunch of cron jobs that run on my infra

This is documented on my blog at https://writing.natwelch.com/post/699.

Right now, the following messages are sent to this job at least once during the time period.

```
{"job": "goodreads"}
{"job": "minute"}
{"job": "pinboard"}
{"job": "random-tweets"}
{"job": "update"}
{"job": "uptime"}
{"job": "user-tweets"}
```

The following is disabled:

```
{"job": "spider"}
```

These can be configured at https://console.cloud.google.com/cloudscheduler?project=icco-cloud
