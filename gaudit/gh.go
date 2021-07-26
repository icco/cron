package gaudit

import (
	"context"
	"fmt"

	"github.com/google/go-github/v37/github"
	"github.com/icco/cron/shared"
	"golang.org/x/oauth2"
)

type Config struct {
	shared.Config

	User        string
	GithubToken string
}

func (c *Config) CheckRepos(ctx context.Context) error {
	client := GithubClient(ctx, c.GithubToken)
	opt := &github.RepositoryListOptions{Type: "owner", Sort: "updated", Direction: "desc"}

	repos, _, err := client.Repositories.List(ctx, c.User, opt)
	if err != nil {
		return err
	}

	for _, r := range repos {
		c.Log.Infow(fmt.Sprintf("%s/%s", user, r.GetName()), "repo", r)
	}

	return nil
}

func GithubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}
