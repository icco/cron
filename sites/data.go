package sites

// SiteMap defines a site we deploy.
type SiteMap struct {
	Host       string
	Owner      string
	Repo       string
	Deployment string
	Branch     string
}

// All contains a list of all domains I update from my code.
var All = []SiteMap{
	{
		Host:       "aniplaxt.natwelch.com",
		Owner:      "icco",
		Repo:       "aniplaxt",
		Deployment: "aniplaxt",
		Branch:     "master",
	},
	{
		Host:       "cacophony.natwelch.com",
		Owner:      "icco",
		Repo:       "cacophony",
		Deployment: "cacophony",
		Branch:     "master",
	},
	{
		Host:       "chartopia.app",
		Owner:      "icco",
		Repo:       "charts",
		Deployment: "charts",
		Branch:     "master",
	},
	{
		Host:       "code.natwelch.com",
		Owner:      "icco",
		Repo:       "code.natwelch.com",
		Deployment: "code",
		Branch:     "main",
	},
	{
		Host:       "cron.natwelch.com",
		Owner:      "icco",
		Repo:       "cron",
		Deployment: "cron",
		Branch:     "main",
	},
	{
		Host:       "etu.natwelch.com",
		Owner:      "icco",
		Repo:       "etu",
		Deployment: "etu",
		Branch:     "main",
	},
	{
		Host:       "interview.natwelch.com",
		Owner:      "icco",
		Repo:       "interview",
		Deployment: "interview",
		Branch:     "main",
	},
	{
		Host:       "gotak.app",
		Owner:      "icco",
		Repo:       "gotak",
		Deployment: "gotak",
		Branch:     "main",
	},
	{
		Host:       "graphql.natwelch.com",
		Owner:      "icco",
		Repo:       "graphql",
		Deployment: "graphql",
		Branch:     "master",
	},
	{
		Host:       "hello.natwelch.com",
		Owner:      "icco",
		Repo:       "hello",
		Deployment: "hello",
		Branch:     "master",
	},
	{
		Host:       "inspiration.natwelch.com",
		Owner:      "icco",
		Repo:       "inspiration",
		Deployment: "inspiration",
		Branch:     "master",
	},
	{
		Host:       "kyoudai.industries",
		Owner:      "trav3711",
		Repo:       "kyoudai.industries",
		Deployment: "kyoudai",
		Branch:     "main",
	},
	{
		Host:       "life.natwelch.com",
		Owner:      "icco",
		Repo:       "lifeline",
		Deployment: "life",
		Branch:     "master",
	},
	{
		Host:       "melandnat.com",
		Owner:      "icco",
		Repo:       "melandnat.com",
		Deployment: "melandnat",
		Branch:     "master",
	},
	{
		Host:       "numbers.natwelch.com",
		Owner:      "icco",
		Repo:       "numbers",
		Deployment: "numbers",
		Branch:     "master",
	},
	{
		Host:       "natwelch.com",
		Owner:      "icco",
		Repo:       "natwelch.com",
		Deployment: "natwelch",
		Branch:     "main",
	},
	{
		Host:       "photos.natwelch.com",
		Owner:      "icco",
		Repo:       "photos",
		Deployment: "photos",
		Branch:     "master",
	},
	{
		Host:       "postmortems.app",
		Owner:      "icco",
		Repo:       "postmortems",
		Deployment: "postmortems",
		Branch:     "master",
	},
	{
		Host:       "quotes.natwelch.com",
		Owner:      "icco",
		Repo:       "crackquotes",
		Deployment: "quotes",
		Branch:     "master",
	},
	{
		Host:       "relay.natwelch.com",
		Owner:      "icco",
		Repo:       "relay",
		Deployment: "relay",
		Branch:     "master",
	},
	{
		Host:       "reportd.natwelch.com",
		Owner:      "icco",
		Repo:       "reportd",
		Deployment: "reportd",
		Branch:     "master",
	},
	{
		Host:       "realworldsre.com",
		Owner:      "icco",
		Repo:       "realworldsre.com",
		Deployment: "realworldsre",
		Branch:     "main",
	},
	{
		Host:       "resume.natwelch.com",
		Owner:      "icco",
		Repo:       "resume",
		Deployment: "resume",
		Branch:     "master",
	},
	{
		Host:       "sadnat.com",
		Owner:      "icco",
		Repo:       "sadnat.com",
		Deployment: "sadnat",
		Branch:     "master",
	},
	{
		Host:       "tab-archive.app",
		Owner:      "icco",
		Repo:       "tab-archive",
		Deployment: "tabs",
		Branch:     "main",
	},
	{
		Host:       "traviscwelch.com",
		Owner:      "trav3711",
		Repo:       "traviscwelch.com",
		Deployment: "traviscwelch",
		Branch:     "master",
	},
	{
		Host:       "validator.natwelch.com",
		Owner:      "icco",
		Repo:       "validator",
		Deployment: "validator",
		Branch:     "main",
	},
	{
		Host:       "walls.natwelch.com",
		Owner:      "icco",
		Repo:       "wallpapers",
		Deployment: "walls",
		Branch:     "master",
	},
	{
		Host:       "writing.natwelch.com",
		Owner:      "icco",
		Repo:       "writing",
		Deployment: "writing",
		Branch:     "master",
	},
}
