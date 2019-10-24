package updater

import (
	"context"
	"fmt"

	"github.com/google/go-github/v28/github"
	"github.com/icco/cron/sites"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type Config struct {
	Log         *logrus.Logger
	GithubToken string
}

var (
	c *Config
)

func UpdateWorkspaces(ctx context.Context, conf *Config) {
	repoFmt := "gcr.io/icco-cloud/%s:%s"
	c = conf

	for _, r := range sites.All {
		sha, err := c.GetSHA(ctx, r.Owner, r.Repo)
		if _, ok := err.(*github.RateLimitError); ok {
			c.Log.WithContext(ctx).WithError(err).Warn("hit rate limit")
			break
		}

		if sha == "" {
			c.Log.WithContext(ctx).WithFields(logrus.Fields{"owner": r.Owner, "repo": r.Repo}).Error("SHA is empty")
			break
		}

		repo := fmt.Sprintf(repoFmt, r.Repo, sha)
		err = UpdateKube(ctx, r, repo)
		if err != nil {
			c.Log.WithError(err).WithContext(ctx).Fatal(err)
		}
	}
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
		result, getErr := deploymentsClient.Get(r.Deployment, metav1.GetOptions{})
		if getErr != nil {
			return (fmt.Errorf("Failed to get latest version of Deployment: %v", getErr))
		}

		result.Spec.Template.Spec.Containers[0].Image = pkg
		_, updateErr := deploymentsClient.Update(result)
		return updateErr
	})
	if retryErr != nil {
		return fmt.Errorf("Update failed: %v", retryErr)
	}

	c.Log.WithContext(ctx).WithFields(logrus.Fields{
		"package":    pkg,
		"deployment": r.Deployment,
	}).Debug("updated deployment")

	return nil
}

func (c *Config) GetSHA(ctx context.Context, owner string, repo string) (string, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.GithubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	branch, _, err := client.Repositories.GetBranch(ctx, owner, repo, "master")
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
