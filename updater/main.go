package updater

import (
	"context"
	"fmt"

	"github.com/google/go-github/v26/github"
	"github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type Config struct {
	Log *logrus.Logger
}

type SiteMap struct {
	Host       string
	Owner      string
	Repo       string
	Deployment string
}

var (
	c *Config

	// AllSites contains a list of all domains I update from my code.
	AllSites = []SiteMap{
		{
			Host:       "cacophony.natwelch.com",
			Owner:      "icco",
			Repo:       "cacophony",
			Deployment: "cacophony",
		},
		{
			Host:       "chartopia.app",
			Owner:      "icco",
			Repo:       "charts",
			Deployment: "charts",
		},
		{
			Host:       "code.natwelch.com",
			Owner:      "icco",
			Repo:       "code.natwelch.com",
			Deployment: "code",
		},
		{
			Host:       "cron.natwelch.com",
			Owner:      "icco",
			Repo:       "cron",
			Deployment: "cron",
		},
		{
			Host:       "etu.natwelch.com",
			Owner:      "icco",
			Repo:       "etu",
			Deployment: "etu",
		},
		{
			Host:       "gotak.app",
			Owner:      "icco",
			Repo:       "gotak",
			Deployment: "gotak",
		},
		{
			Host:       "graphql.natwelch.com",
			Owner:      "icco",
			Repo:       "graphql",
			Deployment: "graphql",
		},
		{
			Host:       "hello.natwelch.com",
			Owner:      "icco",
			Repo:       "hello",
			Deployment: "hello",
		},
		{
			Host:       "inspiration.natwelch.com",
			Owner:      "icco",
			Repo:       "inspiration",
			Deployment: "inspiration",
		},
		{
			Host:       "life.natwelch.com",
			Owner:      "icco",
			Repo:       "lifeline",
			Deployment: "life",
		},
		{
			Host:       "melandnat.com",
			Owner:      "icco",
			Repo:       "melandnat.com",
			Deployment: "melandnat",
		},
		{
			Host:       "natwelch.com",
			Owner:      "icco",
			Repo:       "natwelch.com",
			Deployment: "natwelch",
		},
		{
			Host:       "photos.natwelch.com",
			Owner:      "icco",
			Repo:       "photos",
			Deployment: "photos",
		},
		{
			Host:       "quotes.natwelch.com",
			Owner:      "icco",
			Repo:       "crackquotes",
			Deployment: "quotes",
		},
		{
			Host:       "resume.natwelch.com",
			Owner:      "icco",
			Repo:       "resume",
			Deployment: "resume",
		},
		{
			Host:       "walls.natwelch.com",
			Owner:      "icco",
			Repo:       "wallpapers",
			Deployment: "walls",
		},
		{
			Host:       "writing.natwelch.com",
			Owner:      "icco",
			Repo:       "writing",
			Deployment: "writing",
		},
	}
)

func UpdateWorkspaces(ctx context.Context, conf *Config) {
	repoFmt := "gcr.io/icco-cloud/%s:%s"
	c = conf

	for _, r := range AllSites {
		sha, err := GetSHA(ctx, r.Owner, r.Repo)
		if _, ok := err.(*github.RateLimitError); ok {
			c.Log.WithContext(ctx).WithError(err).Warn("hit rate limit")
			break
		}

		repo := fmt.Sprintf(repoFmt, r.Repo, sha)
		err = UpdateKube(ctx, r, repo)
		if err != nil {
			c.Log.WithError(err).WithContext(ctx).Fatal(err)
		}
	}
}

func UpdateKube(ctx context.Context, r SiteMap, pkg string) error {
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

func GetSHA(ctx context.Context, owner string, repo string) (string, error) {
	client := github.NewClient(nil)
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
