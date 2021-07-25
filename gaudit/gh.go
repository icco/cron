package gaudit

import (
	"context"
	"fmt"

	"github.com/google/go-github/v37/github"
)

func checkRepos(ctx context.Context) error {
	client := github.NewClient(nil)
	opt := &github.RepositoryListOptions{Type: "owner", Sort: "updated", Direction: "desc"}
	user := "icco"

	repos, _, err := client.Repositories.List(ctx, user, opt)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Recently updated repositories by %q: %v", user, github.Stringify(repos))

}
