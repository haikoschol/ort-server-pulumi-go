package keycloak

import (
	"github.com/haikoschol/ort-server-pulumi-go/common"
	pulumiv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	pulumimetav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	corev1 "k8s.io/api/core/v1"
	"os"
	"time"
)

type Cluster struct {
	pulumi.ResourceState

	tlsSecret               *pulumiv1.Secret
	clusterCRDManifest      *yaml.ConfigFile
	realmImportsCRDManifest *yaml.ConfigFile
	operatorManifest        *yaml.ConfigFile
	clusterManifest         *yaml.ConfigFile
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
	err := ctx.RegisterComponentResource("keycloak:Cluster", name, component, opts...)
	if err != nil {
		return nil, err
	}

	component.tlsSecret, err = createTLSSecret(ctx, component)
	if err != nil {
		return nil, err
	}

	component.clusterCRDManifest, err = yaml.NewConfigFile(ctx, "keycloak-crd",
		&yaml.ConfigFileArgs{
			File: "./keycloak/keycloaks.k8s.keycloak.org-v1.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	component.realmImportsCRDManifest, err = yaml.NewConfigFile(ctx, "keycloak-realmimports",
		&yaml.ConfigFileArgs{
			File: "./keycloak/keycloakrealmimports.k8s.keycloak.org-v1.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	component.operatorManifest, err = yaml.NewConfigFile(ctx, "keycloak-operator",
		&yaml.ConfigFileArgs{
			File: "./keycloak/cluster-operator.yaml",
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	if !ctx.DryRun() {
		client, err := common.NewKubernetesClient("ort-server")
		if err != nil {
			return nil, err
		}

		_, err = client.WaitForPods(func() ([]corev1.Pod, error) {
			return client.GetPodsWithLabel("app.kubernetes.io/name=keycloak-operator")
		}, time.Minute)
	}

	component.clusterManifest, err = yaml.NewConfigFile(ctx, "keycloak-cluster",
		&yaml.ConfigFileArgs{
			File: "./keycloak/cluster.yaml",
		},
		pulumi.DependsOn([]pulumi.Resource{
			component.tlsSecret,
			component.clusterCRDManifest,
			component.realmImportsCRDManifest,
		}),
		pulumi.DependsOn([]pulumi.Resource{component.tlsSecret}),
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
	if err != nil {
		return nil, err
	}

	return component, nil
}

func createTLSSecret(ctx *pulumi.Context, component *Cluster) (*pulumiv1.Secret, error) {
	tlsCert, err := os.ReadFile("./keycloak/certificate.pem")
	if err != nil {
		return nil, err
	}

	tlsKey, err := os.ReadFile("./keycloak/key.pem")
	if err != nil {
		return nil, err
	}

	return pulumiv1.NewSecret(
		ctx,
		"keycloak-tls",
		&pulumiv1.SecretArgs{
			Metadata: pulumimetav1.ObjectMetaArgs{
				Name:      pulumi.String("keycloak-tls"),
				Namespace: pulumi.String("ort-server"),
			},
			Type: pulumi.String("kubernetes.io/tls"),
			StringData: pulumi.StringMap{
				"tls.crt": pulumi.String(tlsCert),
				"tls.key": pulumi.String(tlsKey),
			},
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
}
