package postgresql

import (
	"github.com/haikoschol/ort-server-pulumi-go/common"
	pulumiv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	pulumimetav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi-random/sdk/v4/go/random"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type Cluster struct {
	pulumi.ResourceState

	keycloakSecret   *pulumiv1.Secret
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

	var keycloakPassword *random.RandomPassword
	keycloakPassword, component.keycloakSecret, err = createKeycloakSecret(ctx, component)
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

	if !ctx.DryRun() {
		client, err := common.NewKubernetesClient("cnpg-system")
		if err != nil {
			return nil, err
		}

		_, err = client.WaitForPods(func() ([]corev1.Pod, error) {
			return client.GetPodsWithLabel("app.kubernetes.io/name=cloudnative-pg")
		}, time.Minute)
	}

	component.clusterManifest, err = yaml.NewConfigFile(ctx, "postgresql-cluster",
		&yaml.ConfigFileArgs{
			File: "./postgresql/cluster.yaml",
		},
		pulumi.DependsOn([]pulumi.Resource{component.keycloakSecret}),
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	ctx.Export("keycloak-postgresql-password", keycloakPassword.Result)
	return component, nil
}

func createKeycloakSecret(ctx *pulumi.Context, component *Cluster) (*random.RandomPassword, *pulumiv1.Secret, error) {
	password, err := random.NewRandomPassword(
		ctx,
		"keycloak-postgresql-password",
		&random.RandomPasswordArgs{
			Length: pulumi.Int(16),
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, nil, err
	}

	secret, err := pulumiv1.NewSecret(
		ctx,
		"keycloak-tls",
		&pulumiv1.SecretArgs{
			Metadata: pulumimetav1.ObjectMetaArgs{
				Name:      pulumi.String("postgresql-keycloak"),
				Namespace: pulumi.String("ort-server"),
				Labels: pulumi.StringMap{
					"cnpg.io/reload": pulumi.String("true"),
				},
			},
			Type: pulumi.String("kubernetes.io/basic-auth"),
			StringData: pulumi.StringMap{
				"username": pulumi.String("keycloak"),
				"password": password.Result,
			},
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, nil, err
	}

	return password, secret, nil
}
