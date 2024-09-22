package resources

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"
)

func NewBucket(ctx *pulumi.Context, name string, bucketName pulumi.StringOutput, projectId string, region string) (*scaleway.ObjectBucket, error) {
	return scaleway.NewObjectBucket(ctx, name, &scaleway.ObjectBucketArgs{
		ProjectId:    pulumi.String(projectId),
		Name:         bucketName,
		Region:       pulumi.String(region),
		ForceDestroy: pulumi.Bool(true), // TODO: might not work properly to destroy buckets that contains files
	})
}
