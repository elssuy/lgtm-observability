package resources

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumiverse/pulumi-scaleway/sdk/go/scaleway"
)

type ClusterPrivateNetwork struct {
	Name string
}

type ClusterNodePool struct {
	Name     string
	NodeType string
	Size     int
}

type Cluster struct {
	ProjectId      string
	Name           string
	Version        string
	NodePool       []ClusterNodePool
	PrivateNetwork ClusterPrivateNetwork
}

func NewCluster(ctx *pulumi.Context, cluster Cluster) (*scaleway.KubernetesCluster, error) {

	pn, err := scaleway.NewVpcPrivateNetwork(ctx, cluster.PrivateNetwork.Name, &scaleway.VpcPrivateNetworkArgs{
		ProjectId: pulumi.String(cluster.ProjectId),
		Name:      pulumi.String(cluster.PrivateNetwork.Name),
	})
	if err != nil {
		return nil, err
	}

	cl, err := scaleway.NewKubernetesCluster(ctx, cluster.Name, &scaleway.KubernetesClusterArgs{
		ProjectId:                 pulumi.String(cluster.ProjectId),
		Name:                      pulumi.String(cluster.Name),
		Version:                   pulumi.String(cluster.Version),
		Cni:                       pulumi.String("cilium"),
		PrivateNetworkId:          pn.ID(),
		DeleteAdditionalResources: pulumi.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	for _, nodepool := range cluster.NodePool {
		_, err = scaleway.NewKubernetesNodePool(ctx, nodepool.Name, &scaleway.KubernetesNodePoolArgs{
			ClusterId: cl.ID(),
			Name:      pulumi.String(nodepool.Name),
			NodeType:  pulumi.String(nodepool.NodeType),
			Size:      pulumi.Int(nodepool.Size),
		})
		if err != nil {
			return nil, err
		}
	}

	return cl, nil

}
