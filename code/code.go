package code

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/go-github/v34/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
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

	return nil
}

// FetchCommits gets all commits from githubarchive.org for a user at an hour.
func (cfg *Config) FetchCommits(ctx context.Context, year int, month time.Month, day, hour int) ([]*Commit, error) {
	t := time.Date(year, month, day, hour, 0, 0, 0, time.UTC)
	u := fmt.Sprintf("https://data.githubarchive.org/%s.json.gz", t.Format("2006-01-02-<15>"))

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
	cfg.Log.Debugw("got archive response", "url", u, "response", resp)

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

		cfg.Log.Debug("got data", "github", gh)
		if gh.Type == "PushEvent" && gh.Actor.Login == cfg.User {
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

// GithubClient creates a new GithubClient.
func GithubClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc)
}

func RateLimited(err error, log *zap.SugaredLogger) bool {
	_, ok := err.(*github.RateLimitError)
	if ok {
		log.Warnw("hit rate limit", zap.Error(err))
	}

	return ok
}

//   # This makes sure all commits from a repo's commit history are in the
//   # database and have the correct data.
//   #
//   # NOTE: This will probably blow out your github request quota.
//   def self.update_repo user, repo, client = nil, check = true
//     commits = client.commits("#{user}/#{repo}").delete_if {|commit| commit.is_a? String }
//     commited_commits = Commit.where(:repo => repo).group(:repo).count.values.first.to_i
//     if check or commited_commits < commits.count
//       logger.info "#{user}/#{repo} has #{commited_commits} commited commits, but needs #{commits.count}."
//       commits.shuffle.each do |commit|
//         Commit.factory user, repo, commit['sha'], client, check
//       end
//       commited_commits = Commit.where(:repo => repo).group(:repo).count.values.first.to_i
//       logger.info "#{user}/#{repo} has #{commited_commits} commited commits, which is now enough (#{commits.count}). Done."
//     else
//       logger.info "#{user}/#{repo} has #{commited_commits} commited commits, which is enough (#{commits.count}). Skipping."
//     end
//
//     return commited_commits
//   end

//   # This creates a Commit.
//   #
//   # NOTE: repo + sha are supposed to be unique, so if those two already exist,
//   # but the user is wrong, we'll try and update (if check? is true).
//   def self.factory user, repo, sha, client = nil, check = false
//     if client.nil?
//       client = Octokit::Client.new({})
//     end
//
//     commit = Commit.where(:repo => repo, :sha => sha).first_or_initialize
//     if !commit.new_record? and !commit.changed? and !check
//       logger.push "#{user}/#{repo}##{sha} already exists as #{commit.inspect}.", :info
//       # We return nil for better logging above.
//       return nil
//     end
//
//     # No need to check.
//     if check and !commit.new_record? and commit.user.eql? user
//       return nil
//     end
//
//     raise "Github ratelimit remaining #{client.ratelimit.remaining} of #{client.ratelimit.limit} is not enough." if client.ratelimit.remaining < 2
//
//     begin
//       gh_commit = client.commit("#{user}/#{repo}", sha)
//
//       # This is to prevent counting repos I just forked and didn't do any work
//       # in. A few commits will still slip through thought that don't belong to
//       # me. I don't know why.
//       blob = gh_commit[:commit]
//       if blob[:author]
//         if blob[:author][:email]
//           found_user = self.lookup_user blob[:author][:email], client
//           if !found_user.nil?
//             commit.user = found_user
//           else
//             logger.warn "No login found for #{repo}##{sha}: #{blob[:author][:email]}. Using 'null'."
//             commit.user = "null"
//           end
//         else
//           logger.warn "No email found in author blob for #{repo}##{sha}: #{blob[:author].inspect}."
//         end
//       elsif gh_commit.author
//         if gh_commit.author.login
//           commit.user = gh_commit.author.login
//         elsif gh_commit.author.email
//           found_user = self.lookup_user gh_commit.author.email, client
//           if !found_user.nil?
//             commit.user = found_user
//           else
//             logger.warn "No login found for #{repo}##{sha}: #{gh_commit.author.email.inspect}. Using 'null'."
//             commit.user = "null"
//           end
//         else
//           logger.warn "No email or login found for #{repo}##{sha}: gh_commit.author: #{gh_commit.author.inspect}"
//         end
//       else
//         logger.warn "No author found for #{repo}##{sha}: gh_commit: #{gh_commit.inspect}"
//       end
//
//       commit.repo = repo
//       commit.sha = sha
//
//       create_date = gh_commit.commit.author.date
//       if create_date.is_a? String
//         commit.created_on = DateTime.iso8601(create_date)
//       else
//         commit.created_on = create_date
//       end
//
//       if commit.valid?
//         commit.save
//         return commit
//       else
//         logger.push("Error Saving Commit #{user}/#{repo}:#{commit.sha}: #{commit.errors.messages.inspect}", :error)
//         return nil
//       end
//     rescue Octokit::NotFound
//       logger.push("Error Saving Commit #{user}/#{repo}:#{sha}: 404", :warn)
//     end
//   end

//   # Lookup a user by email and return their username. Caches locally.
//   def self.lookup_user email, client = nil
//     if client.nil?
//       client = Octokit::Client.new({})
//     end
//
//     # Shit isn't cached, do the API call (Ratelimit is 20 calls per minute).
//     response = client.search_users email
//     user = nil
//     if response[:total_count] != 1
//       logger.warn "Inconsistent number of results for #{email.inspect}: #{response.inspect}. Setting to null."
//       user = "null"
//     else
//       user = response[:items][0][:login]
//     end
//
//     return user
//   end
