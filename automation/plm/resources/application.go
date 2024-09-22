package resources

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"
)

func NewBucketApplication(ctx *pulumi.Context, name string, projectId string) (*scaleway.IamApiKey, error) {

	app, err := scaleway.NewIamApplication(ctx, name, &scaleway.IamApplicationArgs{
		Name:        pulumi.String(name),
		Description: pulumi.String("Application to manage bucket for LGTM Stack"),
	})
	if err != nil {
		return nil, err
	}

	_, err = scaleway.NewIamPolicy(ctx, "lgtm-object-storag-rw", &scaleway.IamPolicyArgs{
		Description:   pulumi.String("RW permission to project bucket for LGTM stack"),
		ApplicationId: app.ID(),
		Rules: scaleway.IamPolicyRuleArray{
			scaleway.IamPolicyRuleArgs{
				ProjectIds: pulumi.StringArray{
					pulumi.String(projectId),
				},
				PermissionSetNames: pulumi.StringArray{
					pulumi.String("ObjectStorageFullAccess"),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	apiKey, err := scaleway.NewIamApiKey(ctx, "lgtm", &scaleway.IamApiKeyArgs{
		ApplicationId:    app.ID(),
		DefaultProjectId: pulumi.String(projectId),
	})
	if err != nil {
		return nil, err
	}

	return apiKey, nil
}
