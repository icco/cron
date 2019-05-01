package updater

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

type SiteMap struct {
	Host       string
	Owner      string
	Repo       string
	Deployment string
}

var siteMaps = []SiteMap{
	SiteMap{
		Host:       "cacophony.natwelch.com",
		Owner:      "icco",
		Repo:       "cacophony",
		Deployment: "cacophony",
	},
	SiteMap{
		Host:       "chartopia.app",
		Owner:      "icco",
		Repo:       "charts",
		Deployment: "charts",
	},
	SiteMap{
		Host:       "cron.natwelch.com",
		Owner:      "icco",
		Repo:       "cron",
		Deployment: "cron",
	},
	SiteMap{
		Host:       "etu.natwelch.com",
		Owner:      "icco",
		Repo:       "etu",
		Deployment: "etu",
	},
	SiteMap{
		Host:       "gotak.app",
		Owner:      "icco",
		Repo:       "gotak",
		Deployment: "gotak",
	},
	SiteMap{
		Host:       "graphql.natwelch.com",
		Owner:      "icco",
		Repo:       "graphql",
		Deployment: "graphql",
	},
	SiteMap{
		Host:       "hello.natwelch.com",
		Owner:      "icco",
		Repo:       "hello",
		Deployment: "hello",
	},
	SiteMap{
		Host:       "inspiration.natwelch.com",
		Owner:      "icco",
		Repo:       "inspiration",
		Deployment: "inspiration",
	},
	SiteMap{
		Host:       "life.natwelch.com",
		Owner:      "icco",
		Repo:       "lifeline",
		Deployment: "life",
	},
	SiteMap{
		Host:       "melandnat.com",
		Owner:      "icco",
		Repo:       "melandnat.com",
		Deployment: "melandnat",
	},
	SiteMap{
		Host:       "natwelch.com",
		Owner:      "icco",
		Repo:       "natwelch.com",
		Deployment: "natwelch",
	},
	SiteMap{
		Host:       "quotes.natwelch.com",
		Owner:      "icco",
		Repo:       "crackquotes",
		Deployment: "quotes",
	},
	SiteMap{
		Host:       "resume.natwelch.com",
		Owner:      "icco",
		Repo:       "resume",
		Deployment: "resume",
	},
	SiteMap{
		Host:       "walls.natwelch.com",
		Owner:      "icco",
		Repo:       "wallpapers",
		Deployment: "walls",
	},
	SiteMap{
		Host:       "writing.natwelch.com",
		Owner:      "icco",
		Repo:       "writing",
		Deployment: "writing",
	},
}

func UpdateWorkspaces() {
	repoFmt := "gcr.io/icco-cloud/%s:%s"

	for _, r := range siteMaps {
		sha := GetSHA(r.Owner, r.Repo)
		repo := fmt.Sprintf(repoFmt, r.Repo, sha)
		log.Printf(repo)
		err := UpdateKube(r, repo)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func UpdateKube(r SiteMap, pkg string) error {
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
	log.Print("Updated deployment...")

	return nil
}

func GetSHA(owner string, repo string) string {
	sha := &struct {
		SHA string `json:"sha"`
	}{}

	apiEndpoint := "https://api.github.com/repos/%s/%s/commits/master"
	resp, err := http.Get(fmt.Sprintf(apiEndpoint, owner, repo))
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(body, sha)
	if err != nil {
		log.Fatal(err)
	}

	if sha.SHA == "" {
		log.Fatalf("github.com/%s/%s is not a valid repo", owner, repo)
	}

	return sha.SHA
}
