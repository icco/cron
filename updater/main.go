package updater

import (
	"context"
	"fmt"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"github.com/icco/cron/sites"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

// Config is a config.
type Config struct {
	Log           *zap.SugaredLogger
	GoogleProject string
}

const (
	deployerFormat = "%s-deployer"
)

// UpdateTriggers updates our build triggers on gcp.
func (cfg *Config) UpdateTriggers(ctx context.Context) error {
	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	var trigs []*cloudbuildpb.BuildTrigger
	req := &cloudbuildpb.ListBuildTriggersRequest{
		ProjectId: cfg.GoogleProject,
	}
	it := c.ListBuildTriggers(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed while listing: %w", err)
		}
		trigs = append(trigs, resp)
	}

	cfg.Log.Debugw("found triggers", "triggers", trigs)

	for _, s := range sites.All {
		for _, t := range trigs {
			if t.Name == s.Deployment {
				if err := cfg.upsertBuildTrigger(ctx, c, s, t.Id); err != nil {
					return err
				}
			}

			if fmt.Sprintf(deployerFormat, s.Deployment) == t.Name {
				if err := cfg.upsertDeployTrigger(ctx, c, s, t.Id); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (cfg *Config) upsertBuildTrigger(ctx context.Context, c *cloudbuild.Client, s sites.SiteMap, existingTriggerID string) error {
	createReq := &cloudbuildpb.CreateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		Trigger: &cloudbuildpb.BuildTrigger{
			BuildTemplate: &cloudbuildpb.BuildTrigger_Filename{},
			Name:          s.Deployment,
			Github: &cloudbuildpb.GitHubEventsConfig{
				Name: s.Repo,
				Event: &cloudbuildpb.GitHubEventsConfig_Push{
					Push: &cloudbuildpb.PushFilter{
						GitRef: &cloudbuildpb.PushFilter_Branch{
							Branch: ".*",
						},
					},
				},
				Owner: s.Owner,
			},
		},
	}

	if existingTriggerID == "" {
		cfg.Log.Infow("creating trigger", "request", createReq)
		if _, err := c.CreateBuildTrigger(ctx, createReq); err != nil {
			return fmt.Errorf("could not create trigger %+v: %w", createReq, err)
		}

		return nil
	}

	updateReq := &cloudbuildpb.UpdateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		TriggerId: existingTriggerID,
		Trigger:   createReq.Trigger,
	}

	cfg.Log.Infow("updating trigger", "request", updateReq)
	if _, err := c.UpdateBuildTrigger(ctx, updateReq); err != nil {
		return fmt.Errorf("could not update trigger %+v: %w", updateReq, err)
	}

	return nil
}

func (cfg *Config) upsertDeployTrigger(ctx context.Context, c *cloudbuild.Client, s sites.SiteMap, existingTriggerID string) error {
	createReq := &cloudbuildpb.CreateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		Trigger: &cloudbuildpb.BuildTrigger{
			BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
				// TODO template
			},
			Name: fmt.Sprintf(deployerFormat, s.Deployment),
			Github: &cloudbuildpb.GitHubEventsConfig{
				Name: s.Repo,
				Event: &cloudbuildpb.GitHubEventsConfig_Push{
					Push: &cloudbuildpb.PushFilter{
						GitRef: &cloudbuildpb.PushFilter_Branch{
							Branch: fmt.Sprintf("^%s$", s.Branch),
						},
					},
				},
				Owner: s.Owner,
			},
		},
	}

	if existingTriggerID == "" {
		cfg.Log.Infow("creating trigger", "request", createReq)
		if _, err := c.CreateBuildTrigger(ctx, createReq); err != nil {
			return fmt.Errorf("could not create trigger %+v: %w", createReq, err)
		}

		return nil
	}

	updateReq := &cloudbuildpb.UpdateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		TriggerId: existingTriggerID,
		Trigger:   createReq.Trigger,
	}

	cfg.Log.Infow("updating trigger", "request", updateReq)
	if _, err := c.UpdateBuildTrigger(ctx, updateReq); err != nil {
		return fmt.Errorf("could not update trigger %+v: %w", updateReq, err)
	}

	return nil
}
