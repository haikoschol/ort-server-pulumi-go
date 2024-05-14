package rabbitmq

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
	err := ctx.RegisterComponentResource("rabbitmq:Cluster", name, component, opts...)
	if err != nil {
		return nil, err
	}

	component.operatorManifest, err = yaml.NewConfigFile(ctx, "rabbitmq-operator",
		&yaml.ConfigFileArgs{
			File: "./rabbitmq/cluster-operator.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	if !ctx.DryRun() {
		client, err := common.NewKubernetesClient("rabbitmq-system")
		if err != nil {
			return nil, err
		}

		_, err = client.WaitForPods(func() ([]corev1.Pod, error) {
			return client.GetPodsWithLabel("app.kubernetes.io/name=rabbitmq-cluster-operator")
		}, time.Minute)
	}

	component.clusterManifest, err = yaml.NewConfigFile(ctx, "rabbitmq-cluster",
		&yaml.ConfigFileArgs{
			File: "./rabbitmq/cluster.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	return component, nil
}
