package updater

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"github.com/icco/cron/sites"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

// Update runs a trigger to update a site.
func (cfg *Config) Update(ctx context.Context, site sites.SiteMap) error {
	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	name := fmt.Sprintf(deployerFormat, site.Deployment)
	trig, err := c.GetBuildTrigger(ctx, &cloudbuildpb.GetBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		TriggerId: name,
	})
	if err != nil {
		return fmt.Errorf("get build trigger %q: %w", name, err)
	}

	op, err := c.RunBuildTrigger(ctx, &cloudbuildpb.RunBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		TriggerId: trig.Id,
		Source: &cloudbuildpb.RepoSource{
			ProjectId: cfg.GoogleProject,
			RepoName:  site.Repo,
			Revision: &cloudbuildpb.RepoSource_BranchName{
				BranchName: site.Branch,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("run build trigger %q: %w", name, err)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	return nil
}

// UpdateRandomSite updates one site picked randomly.
func (cfg *Config) UpdateRandomSite(ctx context.Context) error {
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	s := sites.All[rand.Intn(len(sites.All))]
	return cfg.Update(ctx, s)
}
