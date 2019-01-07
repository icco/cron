# cron

A bunch of cron jobs that run on my infra

Right now, the following messages are sent to this job at least once during the time period.

```
{"job": "hourly"}
{"job": "minute"}
{"job": "five-minute"}
{"job": "fifteen-minute"}
```

These can be configured at https://console.cloud.google.com/cloudscheduler?project=icco-cloud
