package uptime

import (
	"context"
	"sort"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/icco/cron/sites"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/genproto/protobuf/field_mask"
)

type Config struct {
	Log       *logrus.Logger
	ProjectID string
}

var (
	ExtraHosts = []string{
		"archive.natwelch.com",
		"corybooker.com",
		"dcwelch.com",
		"google.com",
		"lydiadehn.com",
		"mood.natwelch.com",
		"newyork.welch.io",
		"timebyping.com",
		"www.natwelch.com",
		"www.traviscwelch.com",
	}
)

// UpdateUptimeChecks makes sure there is an uptime check for all of my
// domains.
func UpdateUptimeChecks(ctx context.Context, c *Config) error {
	hosts := []string{}
	hosts = append(hosts, ExtraHosts...)
	for _, s := range sites.All {
		hosts = append(hosts, s.Host)
	}
	sort.Strings(hosts)

	existingChecks, err := c.listChecks(ctx)
	if err != nil {
		return errors.Wrap(err, "list checks")
	}
	checkHostMap := map[string]string{}

	for _, check := range existingChecks {
		mr := check.GetMonitoredResource()
		host := mr.Labels["host"]
		checkHostMap[host] = check.Name
	}
	c.Log.WithFields(logrus.Fields{
		"hosts":           hosts,
		"existing-checks": checkHostMap,
	}).Debug("hosts to check")

	hostConfigMap := map[string]*monitoringpb.UptimeCheckConfig{}
	for _, host := range hosts {
		if val, ok := checkHostMap[host]; ok {
			cfg, err := c.updateCheck(ctx, host, val)
			if err != nil {
				return errors.Wrapf(err, "update check %s", host)
			}

			c.Log.WithFields(logrus.Fields{"job": "uptime", "host": host}).Debug("updated uptime check")
			hostConfigMap[host] = cfg
		} else {
			cfg, err := c.createCheck(ctx, host)
			if err != nil {
				return errors.Wrapf(err, "create check %s", host)
			}

			c.Log.WithFields(logrus.Fields{"job": "uptime", "host": host}).Debug("created uptime check")
			hostConfigMap[host] = cfg
		}
	}

	c.Log.WithFields(logrus.Fields{"hosts": hostConfigMap}).Debugf("uptime configs")

	return nil
}

func (c *Config) createCheck(ctx context.Context, host string) (*monitoringpb.UptimeCheckConfig, error) {
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
					Path:        "/",
					Port:        443,
					UseSsl:      true,
					ValidateSsl: true,
				},
			},
			Timeout: &duration.Duration{Seconds: 5},
			Period:  &duration.Duration{Seconds: 60},
		},
	}

	return client.CreateUptimeCheckConfig(ctx, req)
}

func (c *Config) listChecks(ctx context.Context) ([]*monitoringpb.UptimeCheckConfig, error) {
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

func (c *Config) updateCheck(ctx context.Context, host, id string) (*monitoringpb.UptimeCheckConfig, error) {
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
			Path:        "/",
			Port:        443,
			UseSsl:      true,
			ValidateSsl: true,
		},
	}
	config.Timeout = &duration.Duration{Seconds: 5}
	config.Period = &duration.Duration{Seconds: 60}
	req := &monitoringpb.UpdateUptimeCheckConfigRequest{
		UptimeCheckConfig: config,
		UpdateMask: &field_mask.FieldMask{
			Paths: []string{"display_name", "http_check", "timeout"},
		},
	}

	return client.UpdateUptimeCheckConfig(ctx, req)
}
