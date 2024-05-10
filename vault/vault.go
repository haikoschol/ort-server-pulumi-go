package vault

import (
	"encoding/json"
	"fmt"
	"github.com/haikoschol/ort-server-pulumi-go/common"
	pulumiv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	pulumimeta1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	corev1 "k8s.io/api/core/v1"
	"os"
	"time"
)

type Cluster struct {
	pulumi.ResourceState

	namespace *pulumiv1.Namespace
	release   *helm.Release
}

type ClusterArgs struct {
	Namespace string
}

func NewCluster(ctx *pulumi.Context, name string, args *ClusterArgs, opts ...pulumi.ResourceOption) (*Cluster, error) {
	component := &Cluster{}

	err := ctx.RegisterComponentResource("vault:Cluster", name, component, opts...)
	if err != nil {
		return nil, err
	}

	nodeConfig, err := os.ReadFile("./vault/node-config.hcl")
	if err != nil {
		return nil, err
	}

	component.namespace, err = pulumiv1.NewNamespace(
		ctx,
		args.Namespace,
		&pulumiv1.NamespaceArgs{
			Metadata: &pulumimeta1.ObjectMetaArgs{
				Name: pulumi.String(args.Namespace),
			},
		},
		pulumi.ResourceOption(pulumi.Parent(component)),
	)
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
			Namespace: component.namespace.Metadata.Name(),
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

	keysAndToken := component.release.Status.ApplyT(func(status helm.ReleaseStatus) ([]string, error) {
		if status.Status != "deployed" {
			return nil, fmt.Errorf("expected vault helm release status to be 'deployed', but it is '%s'", status.Status)
		}

		podLabel := "app.kubernetes.io/name=vault"
		client, err := common.NewKubernetesClient(args.Namespace)
		if err != nil {
			return nil, fmt.Errorf("NewKubernetesClient: %w", err)
		}

		initInfo, err := UnsealVault(podLabel, client)
		if err != nil {
			return nil, fmt.Errorf("UnsealVault: %w", err)
		}

		return append([]string{initInfo.RootToken}, initInfo.UnsealKeys...), nil
	}).(pulumi.StringArrayOutput)

	err = ctx.RegisterResourceOutputs(component, pulumi.Map{
		// FIXME should be marked as secret output, otherwise the data is stored unencrypted in the pulumi state file
		"vault-unseal-keys-and-root-token": keysAndToken, // TODO separate them
	})

	return component, nil
}

func UnsealVault(podLabel string, kc *common.KubernetesClient) (iInfo InitInfo, err error) {
	var pods []corev1.Pod
	pods, err = kc.GetPodsWithLabel(podLabel)
	if err != nil {
		return
	}

	// Take the first pod and make it the leader; init & unseal first.
	leaderPod := &pods[0]
	leaderName := leaderPod.Name
	leaderPod, err = kc.WaitForPod(leaderName, time.Minute)
	if err != nil {
		return
	}

	iInfo, err = initPod(leaderPod, kc)
	if err != nil {
		return
	}

	err = UnsealPod(leaderPod, iInfo.UnsealKeys, kc)
	if err != nil {
		return
	}

	for i := range pods {
		pod := &pods[i]
		if pod.Name == leaderName {
			continue
		}

		pod, err = kc.WaitForPod(pod.Name, time.Minute)
		if err != nil {
			return
		}
		if err = joinPod(pod, leaderName, kc); err != nil {
			return
		}
		if err = UnsealPod(pod, iInfo.UnsealKeys, kc); err != nil {
			return
		}
	}
	return
}

func UnsealPod(pod *corev1.Pod, unsealKeys []string, kc *common.KubernetesClient) error {
	for i, key := range unsealKeys {
		if i >= 3 {
			break
		}

		_, err := kc.PodExec(pod, "vault operator unseal "+key)
		if err != nil {
			return fmt.Errorf("UnsealPod(%s): %w", pod.Name, err)
		}
	}

	return nil
}

type InitInfo struct {
	UnsealKeys []string `json:"unseal_keys_b64"`
	RootToken  string   `json:"root_token"`
}

func initPod(pod *corev1.Pod, kc *common.KubernetesClient) (InitInfo, error) {
	output, err := kc.PodExec(pod, "vault operator init -format=json")
	if err != nil {
		return InitInfo{}, fmt.Errorf("initPod(%s): %w", pod.Name, err)
	}

	return parseInitOutput(output)
}

func joinPod(pod *corev1.Pod, leaderPodName string, kc *common.KubernetesClient) error {
	cmd := fmt.Sprintf("vault operator raft join http://%s.vault-internal:8200", leaderPodName)
	_, err := kc.PodExec(pod, cmd)
	if err != nil {
		return fmt.Errorf("joinPod(%s, %s): %w", pod.Name, leaderPodName, err)
	}

	return nil
}

func parseInitOutput(output string) (InitInfo, error) {
	var initInfo InitInfo
	err := json.Unmarshal([]byte(output), &initInfo)
	return initInfo, err
}
