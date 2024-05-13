package vault

import (
	"encoding/json"
	"fmt"
	"github.com/haikoschol/ort-server-pulumi-go/common"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	corev1 "k8s.io/api/core/v1"
	"time"
)

type unsealer struct {
	pulumi.ResourceState

	unsealKeys []string
	rootToken  string
}

func newUnsealer(ctx *pulumi.Context, name string, opts ...pulumi.ResourceOption) (*unsealer, error) {
	component := &unsealer{}
	err := ctx.RegisterComponentResource("vault:unsealer", name, component, opts...)
	if err != nil {
		return nil, err
	}

	if !ctx.DryRun() {
		client, err := common.NewKubernetesClient("ort-server")
		if err != nil {
			return nil, err
		}

		var pods []corev1.Pod
		podLabel := "app.kubernetes.io/name=vault"
		deadline := time.Now().Add(time.Minute * 2)

		for time.Now().Before(deadline) {
			pods, err = client.GetPodsWithLabel(podLabel)
			if err != nil {
				return nil, err
			}
			if len(pods) > 0 {
				break
			}
			time.Sleep(time.Second)
		}

		initInfo, err := unseal(pods, client)
		if err != nil {
			return nil, err
		}

		component.unsealKeys = initInfo.UnsealKeys
		component.rootToken = initInfo.RootToken
	}

	return component, nil
}

func unseal(pods []corev1.Pod, kc *common.KubernetesClient) (iInfo InitInfo, err error) {
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

	err = unsealPod(leaderPod, iInfo.UnsealKeys, kc)
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
		if err = unsealPod(pod, iInfo.UnsealKeys, kc); err != nil {
			return
		}
	}
	return
}

func unsealPod(pod *corev1.Pod, unsealKeys []string, kc *common.KubernetesClient) error {
	for i, key := range unsealKeys {
		if i >= 3 {
			break
		}

		_, err := kc.PodExec(pod, "vault operator unseal "+key)
		if err != nil {
			return fmt.Errorf("unsealPod(%s): %w", pod.Name, err)
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
