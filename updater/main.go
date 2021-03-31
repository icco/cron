package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"github.com/google/go-github/v28/github"
	"github.com/icco/cron/sites"
	"golang.org/x/oauth2"
	"google.golang.org/api/iterator"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type Config struct {
	Log           *zap.SugaredLogger
	GithubToken   string
	GoogleProject string
}

func UpdateWorkspaces(ctx context.Context, c *Config) error {
	repoFmt := "gcr.io/%s/%s:%s"
	c = conf

	for _, r := range sites.All {
		sha, err := c.GetSHA(ctx, r.Owner, r.Repo, r.Branch)
		if _, ok := err.(*github.RateLimitError); ok {
			c.Log.Warnw("hit rate limit", zap.Error(err))
			break
		}

		if sha == "" {
			c.Log.Errorw("SHA is empty", "owner", r.Owner, "repo", r.Repo, "branch", r.Branch)
			break
		}

		repo := fmt.Sprintf(repoFmt, conf.GoogleProject, r.Repo, sha)
		err = UpdateKube(ctx, r, repo)
		if err != nil {
			c.Log.Errorw("update kube", zap.Error(err))
			return err
		}
	}

	return nil
}

func UpdateKube(ctx context.Context, r sites.SiteMap, pkg string) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		dep, getErr := deploymentsClient.Get(ctx, r.Deployment, metav1.GetOptions{})
		if getErr != nil {
			return (fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
		}

		oldPkg := dep.Spec.Template.Spec.Containers[0].Image
		dep.Spec.Template.Spec.Containers[0].Image = pkg
		_, updateErr := deploymentsClient.Update(ctx, dep, metav1.UpdateOptions{})
		if updateErr != nil {
			return updateErr
		}

		if oldPkg != pkg {
			d, err := json.Marshal(map[string]string{
				"cloud deployment": r.Deployment,
				"old pkg":          oldPkg,
				"new pkg":          pkg,
			})
			if err != nil {
				return err
			}
			_, err = http.Post("https://relay.natwelch.com/hook", "application/json", bytes.NewBuffer(d))
			if err != nil {
				return err
			}
		}

		return nil
	})
	if retryErr != nil {
		return fmt.Errorf("Update failed: %w", retryErr)
	}

	c.Log.Debugw("updated deployment", "package", pkg, "deployment", r.Deployment)

	return nil
}

func UpdateTriggers(ctx context.Context, conf *Config) error {
	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	var trigs []*cloudbuildpb.BuildTrigger
	req := &cloudbuildpb.ListBuildTriggersRequest{
		ProjectId: conf.GoogleProject,
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

	conf.Log.Debugw("found triggers", "triggers", trigs)

	for _, s := range sites.All {
		exists := false
		for _, t := range trigs {
			if t.Name == s.Deployment {
				exists = true
				// TODO: If exists, update.
				break
			}
		}

		if !exists {
			req := &cloudbuildpb.CreateBuildTriggerRequest{
				ProjectId: conf.GoogleProject,
				// https://issuetracker.google.com/issues/173534838
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

			conf.Log.Infow("creating trigger", "request", req)
			if _, err := c.CreateBuildTrigger(ctx, req); err != nil {
				return fmt.Errorf("could not create trigger %+v: %w", req, err)
			}
		}
	}

	return nil
}

func (c *Config) GetSHA(ctx context.Context, owner, repo, mainBranch string) (string, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.GithubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	branch, _, err := client.Repositories.GetBranch(ctx, owner, repo, mainBranch)
	if err != nil {
		return "", err
	}

	if branch != nil {
		if branch.Commit != nil {
			if branch.Commit.SHA != nil {
				return *branch.Commit.SHA, nil
			}
		}
	}

	return "", fmt.Errorf("could not get %s/%s", owner, repo)
}
