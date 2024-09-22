package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"observability-stack/automation/deployments"
	"observability-stack/automation/plm"
	"observability-stack/automation/plm/programs"
)

func main() {

	ctx := context.Background()
	name := "foul"
	// email := "ulysse.fontaine@octo.com"
	stackName := fmt.Sprintf("%s.observability.kube.octo.com", name)

	//
	// Requirements and variable declaration
	//

	// Required EnvVariables
	requiredEnv := []string{
		"PULUMI_BACKEND_URL",
		"SCW_ACCESS_KEY",
		"SCW_SECRET_KEY",
		"SCW_DEFAULT_ORGANIZATION_ID",
		"SCW_DEFAULT_REGION",
		"SCW_DEFAULT_ZONE",
	}
	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("required env variable %s not found, env variables: %s should be set", env, strings.Join(requiredEnv, ", "))
		}
	}

	// Deploy Observability Cluster
	log.Printf("==== Deploying Project ====\n")

	projectStack := programs.NewProjectStackConfig(stackName)
	projectLayer, err := plm.NewLayer[programs.ProjectProgramOutput](ctx, projectStack, programs.ProjectProgram)
	if err != nil {
		log.Fatal(err)
	}

	err = projectLayer.Up(ctx)
	if err != nil {
		log.Fatal(err)
	}

	projectOut, err := projectLayer.GetOutputs(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("==== Deploying Observability Cluster Stack ====\n")
	clusterStack := programs.NewClusterStackConfig(stackName, map[string]string{
		"scwProjectId": projectOut.ProjectId,
	})

	clusterLayer, err := plm.NewLayer[programs.ClusterProgramOutput](ctx, clusterStack, programs.ClusterProgram)
	if err != nil {
		log.Fatal(err)
	}

	if err := clusterLayer.Up(ctx); err != nil {
		log.Fatal(err)
	}

	clusterOut, err := clusterLayer.GetOutputs(ctx)
	if err != nil {
		log.Fatal(err)
	}

	kubeconfig := clusterOut.ClusterConfig[0].ConfigFile

	filename := fmt.Sprintf("kubeconfig-%s.yaml", "admin-cluster")
	err = os.WriteFile(filename, []byte(kubeconfig), 0644)
	if err != nil {
		log.Fatalf("failed to write kubeconfig file: %v", err)
	}

	log.Printf("==== Deploying Bucket Stack ====\n")
	bucketStack := programs.NewBucketStackConfig(stackName, map[string]string{
		"scwProjectId": projectOut.ProjectId,
	})

	bucketLayer, err := plm.NewLayer[programs.BucketProgramOutput](ctx, bucketStack, programs.BucketProgram)
	if err != nil {
		log.Fatal(err)
	}

	if err := bucketLayer.Up(ctx); err != nil {
		log.Fatal(err)
	}

	bucketOutput, err := bucketLayer.GetOutputs(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("==== Deploying GitOps Stack ====\n")
	tld, err := deployments.DeployGitOpsStack(ctx, kubeconfig)
	if err != nil {
		log.Fatalf("failed to deploy gitops stack: %v", err)
	}

	log.Printf("==== Deploying Observability Stack ====\n")
	err = deployments.DeployObservabilityStack(ctx, kubeconfig, deployments.DeployObservabilityStackArgs{
		TLD:                         tld,
		MimirBlockBucketName:        bucketOutput.Buckets["mimir-block"].Name,
		MimirRulerBucketName:        bucketOutput.Buckets["mimir-ruler"].Name,
		MimirAlertManagerBucketName: bucketOutput.Buckets["mimir-alertmanager"].Name,
		LokiBucketName:              bucketOutput.Buckets["loki"].Name,
		TempoBucketName:             bucketOutput.Buckets["tempo"].Name,
		AwsAccessKeyId:              bucketOutput.ApiKey.AccessKey,
		AwsSecretAccessKey:          bucketOutput.ApiKey.SecretKey,
	})
	if err != nil {
		log.Fatalf("could not deploy observability application stack: %v", err)
	}

	log.Printf("==== Deploying Mixins ====\n")
	err = deployments.DeployMixins(ctx, "./hack/jsonnet", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	// Print all informations
	log.Println("==== Outputs ====")
	tableOutput := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tableOutput, "Name\tValue")
	fmt.Fprintf(tableOutput, "tld\t%s\n", tld)
	fmt.Fprintf(tableOutput, "argocd-url\thttp://argocd.%s\n", tld)
	fmt.Fprintf(tableOutput, "grafana-url\thttp://grafana.%s\n", tld)
	fmt.Fprintf(tableOutput, "mimir-block-bucket\t%s\n", bucketOutput.Buckets["mimir-block"])
	fmt.Fprintf(tableOutput, "mimir-ruler-bucket\t%s\n", bucketOutput.Buckets["mimir-ruler"])
	fmt.Fprintf(tableOutput, "mimir-alert-bucket\t%s\n", bucketOutput.Buckets["mimir-alertmanager"])
	fmt.Fprintf(tableOutput, "loki-bucket\t%s\n", bucketOutput.Buckets["loki"])
	fmt.Fprintf(tableOutput, "tempo-bucket\t%s\n", bucketOutput.Buckets["tempo"])
	tableOutput.Flush()
}
