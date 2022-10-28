// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package kubernetes_test

import (
	"context"
	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
)

func Test_Kubelet_client_finds_containers_info_using_TLS_and_client_certificate_from_kubeconfig(t *testing.T) {
	t.Parallel()

	kubelet, _ := singleNodeClusterKubelet(t)

	_, err := kubelet.FindContainers(nil)
	require.NoErrorf(t, err, "finding containers")
}

func Test_Kubelet_client_expects_pod_container_statuses_to_have_populated(t *testing.T) {
	kubelet, clientset := singleNodeClusterKubelet(t)

	ns := withTestNamespace(t, clientset.CoreV1().Namespaces())

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-pod-",
			Labels: map[string]string{
				"key":  "value",
				"key2": "value2",
			},
			Annotations: map[string]string{
				"key":  "value",
				"key2": "value2",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "test-container",
					// Use full image name, so tests also pass when containerd is used as container runtime.
					Image: "nginx:latest",
					// Nginx require to run as root by default, so run unprivileged command instead.
					Args: []string{"tail", "-f", "/dev/null"},
					Ports: []corev1.ContainerPort{
						{
							Name:          "foo",
							ContainerPort: 12345,
						},
					},
				},
			},
		},
	}

	pod = withRunningTestPod(t, clientset.CoreV1().Pods(ns), pod)

	containersInfo, err := kubelet.FindContainers([]string{ns})
	require.NoErrorf(t, err, "finding containers")

	require.Equalf(t, len(containersInfo), 1, "expected only one container")

	containerInfo := containersInfo[0]

	portName := pod.Spec.Containers[0].Ports[0].Name
	t.Run("port_name", func(t *testing.T) {
		t.Parallel()

		require.Contains(t, containerInfo.Ports, portName)
	})

	t.Run("port_number", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, containerInfo.Ports[portName], pod.Spec.Containers[0].Ports[0].ContainerPort)
	})

	t.Run("container_name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, containerInfo.Name, pod.Spec.Containers[0].Name)
	})

	t.Run("container_id", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, containerInfo.ID)
	})

	t.Run("container_image", func(t *testing.T) {
		t.Parallel()

		require.Contains(t, containerInfo.Image, pod.Spec.Containers[0].Image)
	})

	t.Run("container_image_id", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, containerInfo.ImageID)
	})

	t.Run("pod_ip", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, containerInfo.PodIP)
	})

	t.Run("pod_labels", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, len(containerInfo.PodLabels), len(pod.Labels))
	})

	t.Run("pod_annotations", func(t *testing.T) {
		t.Parallel()

		require.GreaterOrEqual(t, len(containerInfo.PodAnnotations), len(pod.Annotations))
	})

	t.Run("pod_name", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, containerInfo.PodName, pod.Name)
	})

	t.Run("pod_node_ip", func(t *testing.T) {
		t.Parallel()

		require.NotEmpty(t, containerInfo.NodeIP)
	})

	t.Run("pod_namespace", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, containerInfo.Namespace, ns)
	})
}

func singleNodeClusterKubelet(t *testing.T) (kubernetes.Kubelet, *k8s.Clientset) {
	t.Helper()

	cfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(utils.HomeDir(), ".kube", "config"))
	require.NoErrorf(t, err, "building config from kubeconfig")

	clientset, err := k8s.NewForConfig(cfg)
	require.NoErrorf(t, err, "creating k8s client")

	nodes, err := clientset.CoreV1().Nodes().List(contextWithDeadline(t), metav1.ListOptions{})
	require.NoErrorf(t, err, "listing nodes")

	require.Equalf(t, len(nodes.Items), 1, "expected only one Node on the cluster")

	clusterURL, err := url.Parse(cfg.Host)
	require.NoErrorf(t, err, "parsing API URL")

	conf := &config.Config{
		Port:    10250,
		Host:    clusterURL.Hostname(),
		TLS:     true,
		Timeout: 5 * 60 * 1000, // 5 minutes in miliseconds
	}

	connector := http.DefaultConnector(clientset, conf, cfg)

	httpClient, err := http.New(connector, http.WithMaxRetries(5))
	require.NoErrorf(t, err, "creating HTTP Client")

	kubelet := kubernetes.New(httpClient, conf)

	return kubelet, clientset
}

const (
	// Arbitrary amount of time to let tests exit cleanly before main process terminates.
	timeoutGracePeriod = 200 * time.Millisecond
	testPrefix         = "test-"
)

func withRunningTestPod(t *testing.T, podClient corev1client.PodInterface, pod *corev1.Pod) *corev1.Pod {
	t.Helper()

	ctx := contextWithDeadline(t)

	pod, err := podClient.Create(ctx, pod, metav1.CreateOptions{})
	require.NoErrorf(t, err, "creating test pod")

	err = wait.PollImmediateUntil(time.Second, func() (bool, error) {
		pod, err = podClient.Get(ctx, pod.Name, metav1.GetOptions{})
		require.NoErrorf(t, err, "getting pod")
		if pod.Status.Phase != corev1.PodRunning {
			t.Log("the pod is still not running")

			return false, nil
		}
		return true, nil
	}, ctx.Done())
	require.NoErrorf(t, err, "waiting for the pod to be running")

	return pod
}

func withTestNamespace(t *testing.T, nsClient corev1client.NamespaceInterface) string {
	t.Helper()

	ctx := contextWithDeadline(t)

	ns, err := nsClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: testPrefix,
		},
	}, metav1.CreateOptions{})
	require.NoErrorf(t, err, "creating test namespace")

	t.Cleanup(func() {
		if err := nsClient.Delete(ctx, ns.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("removing namespace %q: %v", ns.Name, err)
		}
	})

	return ns.Name
}

func contextWithDeadline(t *testing.T) context.Context {
	t.Helper()

	deadline, ok := t.Deadline()
	if !ok {
		return context.Background()
	}

	ctx, cancel := context.WithDeadline(context.Background(), deadline.Truncate(timeoutGracePeriod))

	t.Cleanup(cancel)

	return ctx
}
