package programs

import (
	"observability-stack/automation/plm"
	resource "observability-stack/automation/plm/resources"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
)

type BucketInfo struct {
	Name     string `json:"name"`
	Endpoint string `json:"endpoint"`
	Region   string `json:"region"`
}

type BucketApiKey struct {
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

type BucketProgramOutput struct {
	Buckets map[string]BucketInfo `json:"buckets"`
	ApiKey  BucketApiKey          `json:"apikey"`
}

func NewBucketStackConfig(stackName string, config map[string]string) plm.StackConfig {
	return plm.StackConfig{
		StackName:   stackName,
		ProjectName: "buckets",
		Dependencies: []plm.StackDependency{
			{
				Name:    "scaleway",
				Version: "v1.15.0",
				URL:     "github://api.github.com/pulumiverse",
			},
			{
				Name:    "random",
				Version: "v4.16.5",
				URL:     "github://api.github.com/pulumi",
			},
		},
		Config: config,
	}

}

func BucketProgram(ctx *pulumi.Context) error {

	conf := config.New(ctx, "")
	projectId := conf.Require("scwProjectId")

	// Application for bucket access
	appKey, err := resource.NewBucketApplication(ctx, "lgtm-app", projectId)
	if err != nil {
		return err
	}

	ctx.Export("apikey", pulumi.Map{
		"accessKey": appKey.AccessKey,
		"secretKey": appKey.SecretKey,
	})

	randomId, err := random.NewRandomId(ctx, "bucket", &random.RandomIdArgs{
		ByteLength: pulumi.Int(2),
	})
	if err != nil {
		return err
	}

	bucketNames := map[string]pulumi.StringOutput{
		"mimir-block":        pulumi.Sprintf("mimir-block-%s", randomId.Hex),
		"mimir-ruler":        pulumi.Sprintf("mimir-ruler-%s", randomId.Hex),
		"mimir-alertmanager": pulumi.Sprintf("mimir-alertmanager-%s", randomId.Hex),
		"loki":               pulumi.Sprintf("loki-%s", randomId.Hex),
		"tempo":              pulumi.Sprintf("tempo-%s", randomId.Hex),
	}

	bucketMap := pulumi.Map{} // map[string]BucketInfos{}

	for k, v := range bucketNames {

		b, err := resource.NewBucket(ctx, k, v, projectId, "fr-par")
		if err != nil {
			return err
		}

		bucketMap[k] = pulumi.Map{
			"name":     b.Name,
			"endpoint": b.Endpoint,
			"region":   b.Region,
		}
	}

	ctx.Export("buckets", bucketMap)

	return nil
}
