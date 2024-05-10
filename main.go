package main

import (
	"github.com/haikoschol/ort-server-pulumi-go/postgresql"
	"github.com/haikoschol/ort-server-pulumi-go/vault"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := vault.NewCluster(ctx, "vault", &vault.ClusterArgs{Namespace: "vault"})
		if err != nil {
			return err
		}

		_, err = postgresql.NewCluster(ctx, "cnpg-cluster", nil)
		if err != nil {
			return err
		}

		return nil
	})
}
