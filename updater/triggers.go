package updater

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	cloudbuild "cloud.google.com/go/cloudbuild/apiv1/v2"
	"github.com/icco/cron/sites"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	cloudbuildpb "google.golang.org/genproto/googleapis/devtools/cloudbuild/v1"
)

var (
	deployerFormat = "%s-deploy"
	deployTmpl     = template.Must(template.New("deployment.yaml").Parse(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Deployment }}
  labels:
    app: {{ .Deployment }}
spec:
  replicas: 3
  selector:
    matchLabels:
      app: {{ .Deployment }}
  template:
    metadata:
      labels:
        app: {{ .Deployment }}
        tier: web
    spec:
      containers:
      - name: {{ .Deployment }}
        image: gcr.io/icco-cloud/{{ .Repo }}:latest
        ports:
        - name: appport
          containerPort: 8080
        livenessProbe:
          httpGet:
            path: /healthz
            port: appport
        readinessProbe:
          httpGet:
            path: /healthz
            port: appport
        envFrom:
        - configMapRef:
            name: {{ .Deployment }}-env
`))
	serviceTmpl = template.Must(template.New("service.yaml").Parse(`
apiVersion: v1
kind: Service
metadata:
  name: {{ .Deployment }}-service
  labels:
    app: {{ .Deployment }}
  annotations:
    cloud.google.com/neg: '{"ingress": true}'
spec:
  type: NodePort
  selector:
    app: {{ .Deployment }}
    tier: web
  ports:
  - port: 8080
    targetPort: 8080
`))
	hpaTmpl = template.Must(template.New("hpa.yaml").Parse(`
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  labels:
    app: {{ .Deployment }}
  name: {{ .Deployment }}-hpa
  namespace: default
spec:
  maxReplicas: 5
  metrics:
  - resource:
      name: cpu
      target:
        averageUtilization: 80
        type: Utilization
    type: Resource
  minReplicas: 1
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ .Deployment }}
`))
)

// UpdateTriggers updates our build triggers on gcp.
func (cfg *Config) UpdateTriggers(ctx context.Context) error {
	c, err := cloudbuild.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}

	trigs := map[string]*cloudbuildpb.BuildTrigger{}
	req := &cloudbuildpb.ListBuildTriggersRequest{
		ProjectId: cfg.GoogleProject,
	}
	it := c.ListBuildTriggers(ctx, req)
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed while listing: %w", err)
		}
		trigs[resp.Name] = resp
	}

	cfg.Log.Debugw("found triggers", "triggers", trigs)

	for _, s := range sites.All {
		buildTrigger, buildExists := trigs[s.Deployment]
		deployTrigger, deployExists := trigs[fmt.Sprintf(deployerFormat, s.Deployment)]

		buildID := ""
		if buildExists {
			buildID = buildTrigger.Id
		}
		if err := cfg.upsertBuildTrigger(ctx, c, s, buildID); err != nil {
			return err
		}

		deployID := ""
		if deployExists {
			deployID = deployTrigger.Id
		}
		if err := cfg.upsertDeployTrigger(ctx, c, s, deployID); err != nil {
			return err
		}
	}

	return nil
}

func (cfg *Config) upsertBuildTrigger(ctx context.Context, c *cloudbuild.Client, s sites.SiteMap, existingTriggerID string) error {
	createReq := &cloudbuildpb.CreateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		Trigger: &cloudbuildpb.BuildTrigger{
			BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
				Build: &cloudbuildpb.Build{
					Substitutions: map[string]string{
						"_IMAGE_NAME":         fmt.Sprintf("gcr.io/icco-cloud/%s", s.Repo),
						"_DOCKERFILE_DIR":     "",
						"_DOCKERFILE_NAME":    "Dockerfile",
						"_OUTPUT_BUCKET_PATH": fmt.Sprintf("%s_cloudbuild/deploy", cfg.GoogleProject),
					},
					Tags: []string{s.Deployment, "build"},
					Steps: []*cloudbuildpb.BuildStep{
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"build",
								"-t",
								"$_IMAGE_NAME:$COMMIT_SHA",
								".",
								"-f",
								"$_DOCKERFILE_NAME",
							},
							Dir: "$_DOCKERFILE_DIR",
							Id:  "Build",
						},
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"push",
								"$_IMAGE_NAME:$COMMIT_SHA",
							},
							Id: "Push",
						},
					},
				},
			},
			Name: s.Deployment,
			Github: &cloudbuildpb.GitHubEventsConfig{
				Name: s.Repo,
				Event: &cloudbuildpb.GitHubEventsConfig_Push{
					Push: &cloudbuildpb.PushFilter{
						GitRef: &cloudbuildpb.PushFilter_Branch{
							Branch: ".*",
						},
					},
				},
				Owner: s.Owner,
			},
		},
	}

	if existingTriggerID == "" {
		cfg.Log.Infow("creating build trigger", "request", createReq, "site", s)
		if _, err := c.CreateBuildTrigger(ctx, createReq); err != nil {
			return fmt.Errorf("could not create trigger %+v: %w", createReq, err)
		}

		return nil
	}

	updateReq := &cloudbuildpb.UpdateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		TriggerId: existingTriggerID,
		Trigger:   createReq.Trigger,
	}

	cfg.Log.Infow("updating build trigger", "request", updateReq, "site", s)
	if _, err := c.UpdateBuildTrigger(ctx, updateReq); err != nil {
		return fmt.Errorf("could not update trigger %+v: %w", updateReq, err)
	}

	return nil
}

func (cfg *Config) deployYAML(s sites.SiteMap) string {
	var tpl bytes.Buffer
	if err := deployTmpl.Execute(&tpl, s); err != nil {
		cfg.Log.Errorw("couldn't render deployment.yaml", zap.Error(err))
		return ""
	}

	return tpl.String()
}

func (cfg *Config) hpaYAML(s sites.SiteMap) string {
	var tpl bytes.Buffer
	if err := hpaTmpl.Execute(&tpl, s); err != nil {
		cfg.Log.Errorw("couldn't render hpa.yaml", zap.Error(err))
		return ""
	}

	return tpl.String()
}

func (cfg *Config) serviceYAML(s sites.SiteMap) string {
	var tpl bytes.Buffer
	if err := serviceTmpl.Execute(&tpl, s); err != nil {
		cfg.Log.Errorw("couldn't render service.yaml", zap.Error(err))
		return ""
	}

	return tpl.String()
}

func (cfg *Config) upsertDeployTrigger(ctx context.Context, c *cloudbuild.Client, s sites.SiteMap, existingTriggerID string) error {
	createReq := &cloudbuildpb.CreateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		Trigger: &cloudbuildpb.BuildTrigger{
			BuildTemplate: &cloudbuildpb.BuildTrigger_Build{
				Build: &cloudbuildpb.Build{
					Substitutions: map[string]string{
						"_K8S_LABELS":         "",
						"_K8S_ANNOTATIONS":    fmt.Sprintf("gcb-trigger-id=%s", existingTriggerID),
						"_K8S_YAML_PATH":      "/workspace/kubernetes",
						"_IMAGE_NAME":         fmt.Sprintf("gcr.io/icco-cloud/%s", s.Repo),
						"_GKE_LOCATION":       "us-central1",
						"_K8S_APP_NAME":       s.Deployment,
						"_GKE_CLUSTER":        "autopilot-cluster-1",
						"_DOCKERFILE_DIR":     "",
						"_K8S_NAMESPACE":      "default",
						"_DOCKERFILE_NAME":    "Dockerfile",
						"_OUTPUT_BUCKET_PATH": fmt.Sprintf("%s_cloudbuild/deploy", cfg.GoogleProject),
					},
					Tags: []string{"$_K8S_APP_NAME", "deploy"},
					Steps: []*cloudbuildpb.BuildStep{
						{
							Id:   "Write k8s",
							Name: "gcr.io/cloud-builders/curl",
							Args: []string{
								"-c",
								fmt.Sprintf(
									`
                  set -ex;
                  mkdir -p $_K8S_YAML_PATH;
                  echo %q > $_K8S_YAML_PATH/deployment.yaml;
                  echo %q > $_K8S_YAML_PATH/hpa.yaml;
                  echo %q > $_K8S_YAML_PATH/service.yaml;
                  ls -al $_K8S_YAML_PATH
                  cat $_K8S_YAML_PATH/deployment.yaml
                  cat $_K8S_YAML_PATH/hpa.yaml
                  cat $_K8S_YAML_PATH/service.yaml
                  `,
									cfg.deployYAML(s),
									cfg.hpaYAML(s),
									cfg.serviceYAML(s)),
							},
							Entrypoint: "sh",
						},
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"build",
								"-t",
								"$_IMAGE_NAME:$COMMIT_SHA",
								".",
								"-f",
								"$_DOCKERFILE_NAME",
							},
							Dir: "$_DOCKERFILE_DIR",
							Id:  "Build",
						},
						{
							Name: "gcr.io/cloud-builders/docker",
							Args: []string{
								"push",
								"$_IMAGE_NAME:$COMMIT_SHA",
							},
							Id: "Push",
						},
						{
							Name: "gcr.io/cloud-builders/gke-deploy",
							Args: []string{
								"prepare",
								"--filename=$_K8S_YAML_PATH",
								"--image=$_IMAGE_NAME:$COMMIT_SHA",
								"--app=$_K8S_APP_NAME",
								"--version=$COMMIT_SHA",
								"--namespace=$_K8S_NAMESPACE",
								"--label=$_K8S_LABELS",
								"--annotation=$_K8S_ANNOTATIONS,gcb-build-id=$BUILD_ID",
								"--create-application-cr",
								`--links="Build details=https://console.cloud.google.com/cloud-build/builds/$BUILD_ID?project=$PROJECT_ID"`,
								"--output=output",
							},
							Id: "Prepare deploy",
						},
						{
							Name: "gcr.io/cloud-builders/gsutil",
							Args: []string{
								"-c",
								`if [ "$_OUTPUT_BUCKET_PATH" != "" ]; then
                  gsutil cp -r output/suggested gs://$_OUTPUT_BUCKET_PATH/config/$_K8S_APP_NAME/$BUILD_ID/suggested
                  gsutil cp -r output/expanded gs://$_OUTPUT_BUCKET_PATH/config/$_K8S_APP_NAME/$BUILD_ID/expanded
                fi`,
							},
							Id:         "Save configs",
							Entrypoint: "sh",
						},
						{
							Name: "gcr.io/cloud-builders/gke-deploy",
							Args: []string{
								"apply",
								"--filename=output/expanded",
								"--cluster=$_GKE_CLUSTER",
								"--location=$_GKE_LOCATION",
								"--namespace=$_K8S_NAMESPACE",
							},
							Id: "Apply deploy",
						},
					},
				},
			},
			Name: fmt.Sprintf(deployerFormat, s.Deployment),
			Github: &cloudbuildpb.GitHubEventsConfig{
				Name: s.Repo,
				Event: &cloudbuildpb.GitHubEventsConfig_Push{
					Push: &cloudbuildpb.PushFilter{
						GitRef: &cloudbuildpb.PushFilter_Branch{
							Branch: fmt.Sprintf("^%s$", s.Branch),
						},
					},
				},
				Owner: s.Owner,
			},
		},
	}

	if existingTriggerID == "" {
		cfg.Log.Infow("creating deploy trigger", "request", createReq, "site", s)
		if _, err := c.CreateBuildTrigger(ctx, createReq); err != nil {
			return fmt.Errorf("could not create trigger %+v: %w", createReq, err)
		}

		return nil
	}

	updateReq := &cloudbuildpb.UpdateBuildTriggerRequest{
		ProjectId: cfg.GoogleProject,
		TriggerId: existingTriggerID,
		Trigger:   createReq.Trigger,
	}

	cfg.Log.Infow("updating deploy trigger", "request", updateReq, "site", s)
	if _, err := c.UpdateBuildTrigger(ctx, updateReq); err != nil {
		return fmt.Errorf("could not update trigger %+v: %w", updateReq, err)
	}

	return nil
}
