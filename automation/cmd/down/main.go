package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"observability-stack/automation/plm"
	"observability-stack/automation/plm/programs"
)

func main() {

	ctx := context.Background()
	name := "foul"
	// email := "ulysse.fontaine@octo.com"
	stackName := fmt.Sprintf("%s.observability.kube.octo.com", name)

	// Required EnvVariables
	requiredEnv := []string{
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

	log.Printf("==== Getting project stack ====\n")
	projectStack := programs.NewProjectStackConfig(stackName)
	projectLayer, err := plm.NewLayer[programs.ProjectProgramOutput](ctx, projectStack, programs.ProjectProgram)
	if err != nil {
		log.Fatal(err)
	}

	projectOut, err := projectLayer.GetOutputs(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("==== Destroying cluster stack ====\n")
	clusterStack := programs.NewClusterStackConfig(stackName, map[string]string{
		"scwProjectId": projectOut.ProjectId,
	})

	clusterLayer, err := plm.NewLayer[programs.ClusterProgramOutput](ctx, clusterStack, programs.ClusterProgram)
	if err != nil {
		log.Fatal(err)
	}

	if err := clusterLayer.Down(ctx); err != nil {
		log.Fatal(err)
	}

	log.Printf("==== Destroying bucket stack ====\n")
	bucketStack := programs.NewBucketStackConfig(stackName, map[string]string{
		"scwProjectId": projectOut.ProjectId,
	})

	bucketLayer, err := plm.NewLayer[programs.BucketProgramOutput](ctx, bucketStack, programs.BucketProgram)
	if err != nil {
		log.Fatal(err)
	}

	// Delete buckets
	if err := bucketLayer.Down(ctx); err != nil {
		log.Fatal(err)
	}

	log.Printf("==== Destroying project stack ====\n")
	// Delete project
	if err := projectLayer.Down(ctx); err != nil {
		log.Fatal(err)
	}

}
