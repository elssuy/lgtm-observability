package programs

import (
	"observability-stack/automation/plm"

	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"
)

type ProjectProgramOutput struct {
	ProjectId string `json:"projectId"`
}

func NewProjectStackConfig(stackName string) plm.StackConfig {
	return plm.StackConfig{
		StackName:   stackName,
		ProjectName: "project",
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
	}
}

func ProjectProgram(ctx *pulumi.Context) error {
	randomId, err := random.NewRandomId(ctx, "projectId", &random.RandomIdArgs{
		ByteLength: pulumi.Int(2),
	})
	if err != nil {
		return err
	}

	project, err := scaleway.NewAccountProject(ctx, "accountProjectResource", &scaleway.AccountProjectArgs{
		Name: pulumi.Sprintf("lgtm-%s", randomId.Hex),
	})
	if err != nil {
		return err
	}

	ctx.Export("projectId", project.ID())

	return nil

}
