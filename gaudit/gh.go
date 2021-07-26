package gaudit

import (
	"context"
	"fmt"

	"github.com/google/go-github/v37/github"
	"github.com/icco/cron/shared"
)

type Config struct {
	shared.Config
}

func (c *Config) checkRepos(ctx context.Context) error {
	client := github.NewClient(nil)
	opt := &github.RepositoryListOptions{Type: "owner", Sort: "updated", Direction: "desc"}
	user := "icco"

	repos, _, err := client.Repositories.List(ctx, user, opt)
	if err != nil {
		return err
	}

	for _, r := range repos {
		c.Log.Infow(fmt.Sprintf("%s/%s", user, r.GetName()), "repo", r)
	}

	return nil
}
