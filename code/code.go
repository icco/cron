package code

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Config is a basic configuration struct.
type Config struct {
	User        string
	Log         *zap.SugaredLogger
	GithubToken string
}

// Commit is a history of a commit in a github repo.
type Commit struct {
	Repo     string    `json:"repo"`
	User     string    `json:"user"`
	SHA      string    `json:"sha"`
	Datetime time.Time `json:"created_on"`
}

// String returns a string representation of a Commit.
func (c *Commit) String() string {
	return fmt.Sprintf("%s/%s#%s", c.User, c.Repo, c.SHA)
}

// FetchAndSaveCommits gets all commits for the last 24 hours and saves to DB.
func (cfg *Config) FetchAndSaveCommits(ctx context.Context) error {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	var tosave []*Commit
	for i := yesterday; i.Before(now); i.Add(time.Hour) {
		cmts, err := cfg.FetchCommits(ctx, i.Year(), i.Month(), i.Day(), i.Hour())
		if err != nil {
			return fmt.Errorf("get %q: %w", i, err)
		}

		tosave = append(tosave, cmts...)
	}

	for _, c := range tosave {
		if err := cfg.Save(ctx, c); err != nil {
			log.Errorw("could not save commit", "commit", c, zap.Error(err))
		}
	}

	return nil
}

// FetchCommits gets all commits from githubarchive.org for a user at an hour.
func (cfg *Config) FetchCommits(ctx context.Context, year int, month time.Month, day, hour int) ([]*Commit, error) {
	t := time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
	u := fmt.Sprintf("https://data.githubarchive.org/%s.json.gz", t.Format("2006-01-02-15"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "icco-cron/1.0")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get archive %q: %w", u, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get archive %q: got %s", u, resp.Status)
	}

	rdr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("new gzip reader: %w", err)
	}
	defer rdr.Close()
	defer resp.Body.Close()

	jd := json.NewDecoder(rdr)
	var data []*Commit
	for jd.More() {
		var gh Github
		if err := jd.Decode(&gh); err != nil {
			return nil, fmt.Errorf("decode json: %w", err)
		}

		if gh.Type == "PushEvent" && gh.Actor.Login == cfg.User {
			cfg.Log.Debugw("got filtered data", "github", gh)
			repo := gh.Repo.Name
			for _, c := range gh.Payload.Commits {
				user, err := cfg.GetUserByEmail(ctx, c.Author.Email)
				if err != nil {
					cfg.Log.Errorw("geting user", zap.Error(err))
					continue
				}

				data = append(data, &Commit{
					Repo: repo,
					SHA:  c.Sha,
					User: user,
					// TODO: Get time
				})
			}
		}
	}

	return data, nil
}

// GetUserByEmail returns a user based on their email.
func (cfg *Config) GetUserByEmail(ctx context.Context, email string) (string, error) {
	client := GithubClient(ctx, cfg.GithubToken)

	result, _, err := client.Search.Users(ctx, email, nil)
	if err != nil {
		if !RateLimited(err, cfg.Log) {
			return "", fmt.Errorf("finding user: %w", err)
		}
	}

	if len(result.Users) == 0 {
		return "", fmt.Errorf("no users found")
	}

	return *result.Users[0].Login, nil
}

// Save saves a commit.
func (cfg *Config) Save(ctx context.Context, commit *Commit) error {
	b, err := json.Marshal(commit)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://code.natwelch.com/save",
		bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("get archive %q: got %s", u, resp.Status)
	}

	return nil
}
