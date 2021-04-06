package code

// class Commit <  ActiveRecord::Base
//   validates :user, :presence => true
//   validates :repo, :presence => true
//   validates :sha, :presence => true, :uniqueness => {:scope => [:repo]}
//
//   def to_s
//     "#{user}/#{repo}##{sha}"
//   end
//
//   # Grabs the commit log from github archive for the specified hour, parses
//   # that and then saves all commits pushed by the USER to the database.
//   def self.fetchAllForTime day, month, year, hour, client = nil
//     require "open-uri"
//     require "zlib"
//
//
//     # Simple error checking.
//     return nil if hour < 0 or hour > 23
//     return nil if day < 1 or day > 31
//     return nil if month < 1 or month > 12
//
//     date = "#{"%4d" % year}-#{"%02d" % month}-#{"%02d" % day}-#{hour}"
//     uri = URI.parse "http://data.githubarchive.org/#{date}.json.gz"
//     Oj.default_options = {symbolize_names: true}
//
//     logger.info "Grabbing #{date} for #{USER.inspect}"
//
//     begin
//       uri.open do |gz|
//         js = Zlib::GzipReader.new(gz).read
//         Oj.load(js) do |event|
//           if event[:actor] == USER
//             logger.debug "Found Event: #{event.inspect}."
//             if event[:type] == "PushEvent"
//               repo = event[:repository][:name]
//               event[:payload][:shas].each do |commit|
//                 sha = commit[0]
//                 user = self.lookup_user commit[1], client
//                 if !user.nil?
//                   ret = self.factory user, repo, sha, client
//                   if !ret.nil?
//                     logger.info "Inserted #{ret}."
//                   end
//                 end
//               end
//             end
//           end
//         end
//       end
//     rescue Timeout::Error
//       logger.warn "The request for #{uri} timed out...skipping."
//     rescue OpenURI::HTTPError => e
//       logger.warn "The request for #{uri} returned an error. #{e.message}"
//     end
//   end
//
//   # This makes sure all commits from a repo's commit history are in the
//   # database and have the correct data.
//   #
//   # NOTE: This will probably blow out your github request quota.
//   def self.update_repo user, repo, client = nil, check = true
//     if client.nil?
//       client = Octokit::Client.new({})
//     end
//
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
//
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
//
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
// end
