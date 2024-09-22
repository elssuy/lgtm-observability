package programs

import (
	"observability-stack/automation/plm"
	resource "observability-stack/automation/plm/resources"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type ClusterConfiguration struct {
	ClusterCaCertificate string `json:"clusterCaCertificate"`
	ConfigFile           string `json:"configFile"`
	Host                 string `json:"host"`
	Token                string `json:"token"`
}

type ClusterProgramOutput struct {
	ClusterConfig []ClusterConfiguration `json:"kubeconfigs"`
}

func NewClusterStackConfig(stackName string, config map[string]string) plm.StackConfig {
	return plm.StackConfig{
		StackName:   stackName,
		ProjectName: "cluster",
		Dependencies: []plm.StackDependency{
			{
				Name:    "scaleway",
				Version: "v1.15.0",
				URL:     "github://api.github.com/pulumiverse",
			},
		},
		Config: config,
	}

}

func ClusterProgram(ctx *pulumi.Context) error {

	conf := config.New(ctx, "")
	projectId := conf.Require("scwProjectId")

	cluster := resource.Cluster{
		ProjectId: projectId,
		Name:      "admin",
		Version:   "1.30.2",
		PrivateNetwork: resource.ClusterPrivateNetwork{
			Name: "admin",
		},
		NodePool: []resource.ClusterNodePool{
			{Name: "1", Size: 3, NodeType: "PRO2_XS"},
		},
	}

	o, err := resource.NewCluster(ctx, cluster)
	if err != nil {
		return err
	}

	ctx.Export("kubeconfigs", o.Kubeconfigs)

	return nil
}
