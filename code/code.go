package code

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/icco/code.natwelch.com/code"
	"github.com/icco/cron/shared"
	"go.uber.org/zap"
)

// Config is a basic configuration struct.
type Config struct {
	shared.Config

	User        string
	GithubToken string
	Cache       *ristretto.Cache
}

// FetchAndSaveCommits gets all commits for the last 24 hours and saves to DB.
func (cfg *Config) FetchAndSaveCommits(ctx context.Context) error {
	if cfg.Cache == nil {
		cache, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: 1e7,     // Num keys to track frequency of (10M).
			MaxCost:     1 << 30, // Maximum cost of cache (1GB).
			BufferItems: 64,      // Number of keys per Get buffer.
		})
		if err != nil {
			return err
		}
		cfg.Cache = cache
	}

	now := time.Now().UTC().Add(-1 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)

	var tosave []*code.Commit
	for i := yesterday; i.Before(now); i = i.Add(time.Hour) {
		cfg.Log.Debugw("fetching one hour of commits", "time", i)
		cmts, err := cfg.FetchCommits(ctx, i.Year(), i.Month(), i.Day(), i.Hour())
		if err != nil {
			return fmt.Errorf("get commits for %q: %w", i, err)
		}

		tosave = append(tosave, cmts...)
	}

	for _, c := range tosave {
		if err := cfg.Save(ctx, c); err != nil {
			cfg.Log.Errorw("could not save commit", "commit", c, zap.Error(err))
		}
	}

	return nil
}

// FetchCommits gets all commits from githubarchive.org for a user at an hour.
func (cfg *Config) FetchCommits(ctx context.Context, year int, month time.Month, day, hour int) ([]*code.Commit, error) {
	t := time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
	if time.Now().Before(t) {
		return nil, fmt.Errorf("cannot fetch commits for the future. %v is after %v", t, time.Now())
	}
	u := fmt.Sprintf("https://data.githubarchive.org/%s.json.gz", t.Format("2006-01-02-15"))

	cfg.Log.Debugw("grabbing one hour of gh archive", "url", u)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Add("User-Agent", "icco-cron/1.0")

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get archive %q: %w", u, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get archive %q != 200: got %s", u, resp.Status)
	}

	rdr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("new gzip reader: %w", err)
	}
	defer rdr.Close()
	defer resp.Body.Close()

	jd := json.NewDecoder(rdr)
	var data []*code.Commit
	for jd.More() {
		var gh Github
		if err := jd.Decode(&gh); err != nil {
			return nil, fmt.Errorf("decode json: %w", err)
		}

		if gh.Type == "PushEvent" && gh.Actor.Login == cfg.User {
			cfg.Log.Debugw("got filtered data", "github", gh)
			repo := gh.Repo.Name
			for _, c := range gh.Payload.Commits {
				getUser, err := cfg.GetUserByEmail(ctx, c.Author.Email)
				var user string
				if err != nil {
					user = cfg.User
					cfg.Log.Debugw("could not get user", "commit", c, "author", c.Author.Email, zap.Error(err))
				} else {
					user = getUser
				}

				data = append(data, &code.Commit{
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
	user, ok := cfg.Cache.Get(email)
	if ok {
		return user.(string), nil
	}
	client := code.GithubClient(ctx, cfg.GithubToken)

	result, _, err := client.Search.Users(ctx, email, nil)
	if err != nil {
		if !code.RateLimited(err, cfg.Log) {
			return "", fmt.Errorf("finding user: %w", err)
		}
	}

	cfg.Log.Debugw("got users", "users", result, "query", email)
	if len(result.Users) == 0 {
		return "", fmt.Errorf("no users found")
	}
	user = *result.Users[0].Login
	cfg.Cache.Set(email, user, 0)

	return user.(string), nil
}

// Save saves a commit.
func (cfg *Config) Save(ctx context.Context, commit *code.Commit) error {
	b, err := json.Marshal(commit)
	if err != nil {
		return fmt.Errorf("could not marshal commit: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://code.natwelch.com/save",
		bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("could not build request: %w", err)
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not save commit: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("save commit %+v: got %s", commit, resp.Status)
	}

	return nil
}
