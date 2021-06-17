package updater

import (
	"context"
	"fmt"
	"time"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"github.com/icco/cron/sites"
	"google.golang.org/api/iterator"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

var (
	deployerFormat = "%s-deploy"
)

// UpdateTriggers updates our build triggers on gcp.
func (cfg *Config) UpdateTriggers(ctx context.Context) error {
	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	trigs := map[string]*cloudbuildpb.BuildTrigger{}
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
		trigs[resp.Name] = resp
	}

	cfg.Log.Debugw("found triggers", "triggers", trigs)

	for _, s := range sites.All {
		buildTrigger, buildExists := trigs[s.Deployment]
		deployTrigger, deployExists := trigs[fmt.Sprintf(deployerFormat, s.Deployment)]

		buildID := ""
		if buildExists {
			buildID = buildTrigger.Id
		}
		if err := cfg.upsertBuildTrigger(ctx, c, s, buildID); err != nil {
			return err
		}

		deployID := ""
		if deployExists {
			deployID = deployTrigger.Id
		}
		if err := cfg.upsertDeployTrigger(ctx, c, s, deployID); err != nil {
			return err
		}
	}

	return nil
}

func (cfg *Config) upsertBuildTrigger(ctx context.Context, c *cloudbuild.Client, s sites.SiteMap, existingTriggerID string) error {
	createReq := &cloudbuildpb.CreateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		Trigger: &cloudbuildpb.BuildTrigger{
			BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
				Build: &cloudbuildpb.Build{
					Timeout: durationpb.New(time.Minute * 20),
					Substitutions: map[string]string{
						"_IMAGE_NAME": fmt.Sprintf("gcr.io/icco-cloud/%s", s.Repo),
					},
					Tags: []string{s.Deployment, "build"},
					Steps: []*cloudbuildpb.BuildStep{
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"build",
								"-t",
								"$_IMAGE_NAME:$COMMIT_SHA",
								".",
								"-f",
								"Dockerfile",
							},
							Id: "Build",
						},
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"push",
								"$_IMAGE_NAME:$COMMIT_SHA",
							},
							Id: "Push SHA",
						},
					},
				},
			},
			Name: s.Deployment,
			Github: &cloudbuildpb.GitHubEventsConfig{
				Name: s.Repo,
				Event: &cloudbuildpb.GitHubEventsConfig_Push{
					Push: &cloudbuildpb.PushFilter{
						GitRef: &cloudbuildpb.PushFilter_Branch{
							Branch: fmt.Sprintf("^%s$", s.Branch),
						},
						InvertRegex: true,
					},
				},
				Owner: s.Owner,
			},
			Tags: []string{"build"},
		},
	}

	if existingTriggerID == "" {
		cfg.Log.Infow("creating build trigger", "request", createReq, "site", s)
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

	cfg.Log.Debugw("updating build trigger", "request", updateReq, "site", s)
	if _, err := c.UpdateBuildTrigger(ctx, updateReq); err != nil {
		return fmt.Errorf("could not update trigger %+v: %w", updateReq, err)
	}

	return nil
}

func (cfg *Config) upsertDeployTrigger(ctx context.Context, c *cloudbuild.Client, s sites.SiteMap, existingTriggerID string) error {
	idStr := "$TRIGGER_NAME"
	if existingTriggerID != "" {
		idStr = existingTriggerID
	}
	createReq := &cloudbuildpb.CreateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		Trigger: &cloudbuildpb.BuildTrigger{
			BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
				Build: &cloudbuildpb.Build{
					Timeout: durationpb.New(time.Minute * 20),
					Substitutions: map[string]string{
						"_PLATFORM":      "managed",
						"_IMAGE_NAME":    fmt.Sprintf("gcr.io/icco-cloud/%s", s.Repo),
						"_DEPLOY_REGION": "us-central1",
						"_SERVICE_NAME":  s.Deployment,
						"_TRIGGER_ID":    idStr,
					},
					Tags: []string{"$_SERVICE_NAME", "deploy"},
					Images: []string{
						"$_IMAGE_NAME:latest",
						"$_IMAGE_NAME:$COMMIT_SHA",
					},
					Steps: []*cloudbuildpb.BuildStep{
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"build",
								"-t",
								"$_IMAGE_NAME:$COMMIT_SHA",
								"-t",
								"$_IMAGE_NAME:latest",
								".",
								"-f",
								"Dockerfile",
							},
							Id: "Build",
						},
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"push",
								"$_IMAGE_NAME:$COMMIT_SHA",
							},
							Id: "Push SHA",
						},
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"push",
								"$_IMAGE_NAME:latest",
							},
							Id: "Push latest",
						},
						{
							Name:       "gcr.io/google.com/cloudsdktool/cloud-sdk:emulators",
							Id:         "VulnScan",
							Entrypoint: "gcloud",
							Args: []string{
								"artifacts",
								"docker",
								"images",
								"scan",
								"$_IMAGE_NAME:$COMMIT_SHA",
								"--remote",
								"--quiet",
							},
						},
						{
							Name: "gcr.io/google.com/cloudsdktool/cloud-sdk:slim",
							Args: []string{
								"run",
								"services",
								"update",
								"$_SERVICE_NAME",
								"--platform=$_PLATFORM",
								"--image=$_IMAGE_NAME:$COMMIT_SHA",
								"--labels=managed-by=gcp-cloud-build-deploy-cloud-run,commit-sha=$COMMIT_SHA,gcb-build-id=$BUILD_ID,gcb-trigger-id=$_TRIGGER_ID",
								"--region=$_DEPLOY_REGION",
								"--quiet",
							},
							Id:         "Deploy",
							Entrypoint: "gcloud",
						},
						{
							Name: "curlimages/curl",
							Args: []string{
								"-svL",
								"-d",
								`"{\"deployed\": \"$_SERVICE_NAME\", \"image\": \"$_IMAGE_NAME:$COMMIT_SHA\"}"`,
								"-X",
								"POST",
								`--header`,
								`Content-Type: application/json`,
								"-f",
								"https://relay.natwelch.com/hook",
							},
							Id: "Notfiy",
						},
					},
				},
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
			Tags: []string{"deploy"},
		},
	}

	if existingTriggerID == "" {
		cfg.Log.Infow("creating deploy trigger", "request", createReq, "site", s)
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

	cfg.Log.Infow("updating deploy trigger", "request", updateReq, "site", s)
	if _, err := c.UpdateBuildTrigger(ctx, updateReq); err != nil {
		return fmt.Errorf("could not update trigger %+v: %w", updateReq, err)
	}

	return nil
}
