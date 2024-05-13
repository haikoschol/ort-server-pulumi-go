package main

import (
	"github.com/haikoschol/ort-server-pulumi-go/keycloak"
	"github.com/haikoschol/ort-server-pulumi-go/postgresql"
	"github.com/haikoschol/ort-server-pulumi-go/vault"
	pulumiv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	pulumimeta1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		namespace, err := pulumiv1.NewNamespace(
			ctx,
			"ort-server",
			&pulumiv1.NamespaceArgs{
				Metadata: &pulumimeta1.ObjectMetaArgs{
					Name: pulumi.String("ort-server"),
				},
			},
		)
		if err != nil {
			return err
		}

		_, err = vault.NewCluster(ctx, "vault-cluster", &vault.ClusterArgs{Namespace: namespace})
		if err != nil {
			return err
		}

		_, err = postgresql.NewCluster(ctx, "cnpg-cluster", &postgresql.ClusterArgs{Namespace: namespace})
		if err != nil {
			return err
		}

		_, err = keycloak.NewCluster(ctx, "keycloak-cluster", &keycloak.ClusterArgs{Namespace: namespace})
		if err != nil {
			return err
		}

		//_, err = rabbitmq.NewCluster(ctx, "rabbitmq-cluster", &rabbitmq.ClusterArgs{Namespace: namespace})
		//if err != nil {
		//	return err
		//}

		return nil
	})
}
