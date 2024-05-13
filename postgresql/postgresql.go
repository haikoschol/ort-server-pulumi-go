package postgresql

import (
	"github.com/haikoschol/ort-server-pulumi-go/common"
	pulumiv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type Cluster struct {
	pulumi.ResourceState

	operatorManifest *yaml.ConfigFile
	clusterManifest  *yaml.ConfigFile
}

type ClusterArgs struct {
	Namespace *pulumiv1.Namespace
}

func NewCluster(
	ctx *pulumi.Context,
	name string,
	args *ClusterArgs,
	opts ...pulumi.ResourceOption,
) (*Cluster, error) {
	component := &Cluster{}
	opts = append(opts, pulumi.DependsOn([]pulumi.Resource{args.Namespace}))
	err := ctx.RegisterComponentResource("cloudnativepg:Cluster", name, component, opts...)
	if err != nil {
		return nil, err
	}

	component.operatorManifest, err = yaml.NewConfigFile(ctx, "cnpg-operator",
		&yaml.ConfigFileArgs{
			File: "./postgresql/cnpg-1.23.1.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	client, err := common.NewKubernetesClient("cnpg-system")
	if err != nil {
		return nil, err
	}

	_, err = client.WaitForPods(func() ([]corev1.Pod, error) {
		return client.GetPodsWithLabel("app.kubernetes.io/name=cloudnative-pg")
	}, time.Minute)

	component.clusterManifest, err = yaml.NewConfigFile(ctx, "postgresql-cluster",
		&yaml.ConfigFileArgs{
			File: "./postgresql/cluster.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	return component, nil
}
