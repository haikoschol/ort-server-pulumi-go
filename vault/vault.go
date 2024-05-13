package vault

import (
	"fmt"
	pulumiv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"os"
)

type Cluster struct {
	pulumi.ResourceState

	release  *helm.Release
	unsealer *unsealer
}

type ClusterArgs struct {
	Namespace *pulumiv1.Namespace
}

func NewCluster(ctx *pulumi.Context, name string, args *ClusterArgs, opts ...pulumi.ResourceOption) (*Cluster, error) {
	component := &Cluster{}
	opts = append(opts, pulumi.DependsOn([]pulumi.Resource{args.Namespace}))
	err := ctx.RegisterComponentResource("vault:Cluster", name, component, opts...)
	if err != nil {
		return nil, err
	}

	nodeConfig, err := os.ReadFile("./vault/node-config.hcl")
	if err != nil {
		return nil, err
	}

	component.release, err = helm.NewRelease(
		ctx,
		"vault",
		&helm.ReleaseArgs{
			Chart: pulumi.String("vault"),
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String("https://helm.releases.hashicorp.com"),
			},
			Version:   pulumi.String("0.27.0"),
			Namespace: args.Namespace.Metadata.Name(),
			ValueYamlFiles: pulumi.AssetOrArchiveArray{
				pulumi.NewFileAsset("./vault/override-values.yml"),
			},
			Values: pulumi.Map{
				"ha": pulumi.Map{
					"raft": pulumi.Map{
						"config": pulumi.String(nodeConfig),
					},
				},
			},
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	component.unsealer, err = newUnsealer(
		ctx,
		"vault-unsealer",
		pulumi.Parent(component.release),
	)
	if err != nil {
		return nil, err
	}

	outputs := make(pulumi.Map)
	for i, key := range component.unsealer.unsealKeys {
		outputs[fmt.Sprintf("vault-unseal-key-%d", i+1)] = pulumi.ToSecret(key)
	}

	outputs["vault-initial-root-key"] = pulumi.ToSecret(component.unsealer.rootToken)
	err = ctx.RegisterResourceOutputs(component, outputs)
	return component, err
}
