package common

import (
	"bytes"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

type KubernetesClient struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
	namespace string
}

func NewKubernetesClient(namespace string) (*KubernetesClient, error) {
	var config *rest.Config
	var err error

	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	config, err = clientcmd.BuildConfigFromFlags("", path.Join(home, ".kube/config"))
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesClient{
		clientset: clientset,
		config:    config,
		namespace: namespace,
	}, nil
}

func (kc *KubernetesClient) PodExecHack(pod *corev1.Pod, command string) (string, error) {
	args := append(
		[]string{
			fmt.Sprintf("--namespace=%s", kc.namespace),
			"exec",
			"--stdin=true",
			"--tty=false",
			pod.Name,
			"--",
		},
		strings.Split(command, " ")...,
	)

	cmd := exec.Command("kubectl", args...)
	var stdout, stderr strings.Builder
	cmd.Stdin = os.Stdin
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("PodExec: %w", err)
	}
	errOutput := stderr.String()
	if errOutput != "" {
		return "", fmt.Errorf("PodExec errOutput: %s", errOutput)
	}

	return stdout.String(), nil
}

func (kc *KubernetesClient) PodExec(pod *corev1.Pod, command string) (string, error) {
	cmd := []string{
		"/bin/sh",
		"-c",
		command,
	}

	req := kc.clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(kc.namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: cmd,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     false,
		}, scheme.ParameterCodec)

	exc, err := remotecommand.NewSPDYExecutor(kc.config, "POST", req.URL())
	if err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer

	err = exc.StreamWithContext(
		context.Background(),
		remotecommand.StreamOptions{
			Stdin:  nil,
			Stdout: &stdout,
			Stderr: &stderr,
			Tty:    false,
		})
	if err != nil {
		return "", err
	}
	errOutput := stderr.String()
	if errOutput != "" {
		return "", fmt.Errorf(`PodExec("%s", "%s") stderr: %s`, pod.Name, cmd, errOutput)
	}

	return stdout.String(), nil
}

func (kc *KubernetesClient) GetPod(name string) (*corev1.Pod, error) {
	return kc.clientset.CoreV1().Pods(kc.namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (kc *KubernetesClient) GetPodsWithLabel(label string) ([]corev1.Pod, error) {
	pods, err := kc.clientset.CoreV1().Pods(kc.namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: label,
	})
	if err != nil {
		return nil, err
	}

	return pods.Items, err
}

func (kc *KubernetesClient) WaitForPod(name string, timeout time.Duration) (*corev1.Pod, error) {
	deadline := time.Now().Add(timeout)

	for {
		pod, err := kc.GetPod(name)
		if err != nil {
			return nil, err
		}

		if kc.IsPodReady(pod) {
			return pod, nil
		} else {
			if time.Now().After(deadline) {
				return nil, fmt.Errorf(
					"timed out waiting for pod %s to be ready within %d seconds",
					name,
					timeout.Seconds(),
				)
			}
			time.Sleep(time.Second * 3)
		}
	}
}

type GetPodsFunc func() ([]corev1.Pod, error)

func (kc *KubernetesClient) WaitForPods(getPods GetPodsFunc, timeout time.Duration) ([]corev1.Pod, error) {
	deadline := time.Now().Add(timeout)

	for {
		pods, err := getPods()
		if err != nil {
			return nil, err
		}

		if kc.PodsReady(pods) {
			return pods, nil
		} else {
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("timed out waiting for pods to be ready within %d seconds", timeout)
			}
			time.Sleep(time.Second * 3)
		}
	}
}

func (kc *KubernetesClient) PodsReady(pods []corev1.Pod) bool {
	if len(pods) == 0 {
		return false
	}

	for _, pod := range pods {
		if !kc.IsPodReady(&pod) {
			return false
		}
	}

	return true
}

func (kc *KubernetesClient) IsPodReady(pod *corev1.Pod) bool {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (kc *KubernetesClient) Clientset() *kubernetes.Clientset {
	return kc.clientset
}
