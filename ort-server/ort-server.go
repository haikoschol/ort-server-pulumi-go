package ortserver

import (
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type ORTServer struct {
	pulumi.ResourceState

	coreManifest         *yaml.ConfigFile
	orchestratorManifest *yaml.ConfigFile
}

type Args struct {
	Namespace *corev1.Namespace
}

func NewORTServer(ctx *pulumi.Context, name string, args *Args, opts ...pulumi.ResourceOption) (*ORTServer, error) {
	component := &ORTServer{}
	opts = append(opts, pulumi.DependsOn([]pulumi.Resource{args.Namespace}))
	err := ctx.RegisterComponentResource("rabbitmq:Cluster", name, component, opts...)
	if err != nil {
		return nil, err
	}

	component.coreManifest, err = yaml.NewConfigFile(ctx, "ort-server-core",
		&yaml.ConfigFileArgs{
			File: "./ort-server/core.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	component.orchestratorManifest, err = yaml.NewConfigFile(ctx, "ort-server-orchestrator",
		&yaml.ConfigFileArgs{
			File: "./ort-server/orchestrator.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
