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
	for _, s := range ExtraHosts {
		hosts = append(hosts, s)
	}
	for _, s := range updater.AllSites {
		hosts = append(hosts, s.Host)
	}

	sort.Strings(hosts)

	existingChecks, err := c.list(ctx)
	if err != nil {
		return err
	}

	for _, check := range existingChecks {
		mr := check.GetMonitoredResource()
		c.Log.Debugf("host found: %+v", mr.Labels["host"])
		i := sort.SearchStrings(hosts, mr.Labels["host"])
		if i >= 0 && i < len(hosts) {
			hosts = remove(hosts, i)
		}
	}

	for _, host := range hosts {
		c.create(ctx, host)
	}

	return nil
}

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

// create creates an example uptime check.
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
	config, err := client.CreateUptimeCheckConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// list is an example of listing the uptime checks in projectID.
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

// update is an example of updating an uptime check. resourceName should be
// of the form `projects/[PROJECT_ID]/uptimeCheckConfigs/[UPTIME_CHECK_ID]`.
//func update(w io.Writer, resourceName, displayName, httpCheckPath string) (*monitoringpb.UptimeCheckConfig, error) {
//	ctx := context.Background()
//	client, err := monitoring.NewUptimeCheckClient(ctx)
//	if err != nil {
//		return nil, fmt.Errorf("NewUptimeCheckClient: %v", err)
//	}
//	defer client.Close()
//	getReq := &monitoringpb.GetUptimeCheckConfigRequest{
//		Name: resourceName,
//	}
//	config, err := client.GetUptimeCheckConfig(ctx, getReq)
//	if err != nil {
//		return nil, fmt.Errorf("GetUptimeCheckConfig: %v", err)
//	}
//	config.DisplayName = displayName
//	config.GetHttpCheck().Path = httpCheckPath
//	req := &monitoringpb.UpdateUptimeCheckConfigRequest{
//		UpdateMask: &field_mask.FieldMask{
//			Paths: []string{"display_name", "http_check.path"},
//		},
//		UptimeCheckConfig: config,
//	}
//	config, err = client.UpdateUptimeCheckConfig(ctx, req)
//	if err != nil {
//		return nil, fmt.Errorf("UpdateUptimeCheckConfig: %v", err)
//	}
//	fmt.Fprintf(w, "Successfully updated %v", resourceName)
//	return config, nil
//}
