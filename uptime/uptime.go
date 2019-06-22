package uptime

import (
	"context"
	"sort"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/icco/cron/updater"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type Config struct {
	Log       *logrus.Logger
	ProjectID string
}

var (
	ExtraHosts = []string{
		"corybooker.com",
	}
)

// UpdateUptimeChecks makes sure there is an uptime check for all of my
// domains.
func UpdateUptimeChecks(ctx context.Context, c *Config) error {
	hosts := []string{}
	hosts = append(hosts, ExtraHosts...)
	for _, s := range updater.AllSites {
		hosts = append(hosts, s.Host)
	}
	sort.Strings(hosts)

	existingChecks, err := c.list(ctx)
	if err != nil {
		return err
	}
	existingHosts := []string{}
	checkHostMap := map[string]string{}

	for _, check := range existingChecks {
		mr := check.GetMonitoredResource()
		host := mr.Labels["host"]
		c.Log.Debugf("host found: %+v", host)
		existingHosts = append(existingHosts, host)
		checkHostMap[host] = check.Name
	}
	sort.Strings(existingHosts)

	for _, host := range hosts {
		i := sort.SearchStrings(existingHosts, host)

		if i >= len(existingHosts) {
			_, err := c.create(ctx, host)
			if err != nil {
				return err
			}
		} else {
			_, err := c.update(ctx, host, checkHostMap[host])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) create(ctx context.Context, host string) (*monitoringpb.UptimeCheckConfig, error) {
	client, err := monitoring.NewUptimeCheckClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	req := &monitoringpb.CreateUptimeCheckConfigRequest{
		Parent: "projects/" + c.ProjectID,
		UptimeCheckConfig: &monitoringpb.UptimeCheckConfig{
			DisplayName: host,
			Resource: &monitoringpb.UptimeCheckConfig_MonitoredResource{
				MonitoredResource: &monitoredres.MonitoredResource{
					Type: "uptime_url",
					Labels: map[string]string{
						"host": host,
					},
				},
			},
			CheckRequestType: &monitoringpb.UptimeCheckConfig_HttpCheck_{
				HttpCheck: &monitoringpb.UptimeCheckConfig_HttpCheck{
					Path:   "/",
					Port:   443,
					UseSsl: true,
				},
			},
			Timeout: &duration.Duration{Seconds: 5},
			Period:  &duration.Duration{Seconds: 60},
		},
	}
	c.Log.Infof("creating %+v", req)
	return client.CreateUptimeCheckConfig(ctx, req)
}

func (c *Config) list(ctx context.Context) ([]*monitoringpb.UptimeCheckConfig, error) {
	client, err := monitoring.NewUptimeCheckClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	req := &monitoringpb.ListUptimeCheckConfigsRequest{
		Parent: "projects/" + c.ProjectID,
	}

	ret := []*monitoringpb.UptimeCheckConfig{}
	it := client.ListUptimeCheckConfigs(ctx, req)
	for {
		config, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		ret = append(ret, config)
	}
	return ret, nil
}

func (c *Config) update(ctx context.Context, host, id string) (*monitoringpb.UptimeCheckConfig, error) {
	client, err := monitoring.NewUptimeCheckClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	getReq := &monitoringpb.GetUptimeCheckConfigRequest{
		Name: id,
	}
	config, err := client.GetUptimeCheckConfig(ctx, getReq)
	if err != nil {
		return nil, err
	}
	config.DisplayName = host
	config.Resource = &monitoringpb.UptimeCheckConfig_MonitoredResource{
		MonitoredResource: &monitoredres.MonitoredResource{
			Type: "uptime_url",
			Labels: map[string]string{
				"host": host,
			},
		},
	}
	config.CheckRequestType = &monitoringpb.UptimeCheckConfig_HttpCheck_{
		HttpCheck: &monitoringpb.UptimeCheckConfig_HttpCheck{
			Path:   "/",
			Port:   443,
			UseSsl: true,
		},
	}
	config.Timeout = &duration.Duration{Seconds: 5}
	config.Period = &duration.Duration{Seconds: 60}
	req := &monitoringpb.UpdateUptimeCheckConfigRequest{
		UptimeCheckConfig: config,
	}
	return client.UpdateUptimeCheckConfig(ctx, req)
}
