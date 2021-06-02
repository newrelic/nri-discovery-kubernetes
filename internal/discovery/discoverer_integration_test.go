// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build integration

package discovery_test

import (
	"context"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/newrelic/nri-discovery-kubernetes/internal/discovery"
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
)

func Test_discovery_when(t *testing.T) {
	t.Parallel()

	cfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(utils.HomeDir(), ".kube", "config"))
	require.NoErrorf(t, err, "building config from kubeconfig")

	k8sClient, err := k8s.NewForConfig(cfg)
	require.NoErrorf(t, err, "creating k8s client")

	clusterURL, err := url.Parse(cfg.Host)

	require.NoErrorf(t, err, "waiting for the pod to be running")

	kubelet, err := kubernetes.NewKubelet(clusterURL.Hostname(), 10250, true, false, time.Minute*5)
	require.NoErrorf(t, err, "creating kubelet client")

	ctx := contextWithDeadline(t)

	nodes, err := k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	require.NoErrorf(t, err, "listing nodes")

	require.Equalf(t, len(nodes.Items), 1, "expected only one Node on the cluster")

	ns, err := k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}, metav1.CreateOptions{})
	require.NoErrorf(t, err, "creating test namespace")

	t.Cleanup(func() {
		if err := k8sClient.CoreV1().Namespaces().Delete(ctx, ns.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("removing namespace %q: %v", ns.Name, err)
		}
	})

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
				},
			},
		},
	}

	podClient := k8sClient.CoreV1().Pods(ns.Name)

	pod, err = podClient.Create(ctx, pod, metav1.CreateOptions{})
	require.NoErrorf(t, err, "creating test pod")

	err = wait.PollImmediateUntil(time.Second, func() (bool, error) {
		pod, err = podClient.Get(ctx, pod.Name, metav1.GetOptions{})
		require.NoErrorf(t, err, "not expecting error getting pod")
		if pod.Status.Phase != corev1.PodRunning {
			t.Log("the pod is still not running")

			return false, nil
		}
		return true, nil
	}, ctx.Done())
	require.NoErrorf(t, err, "waiting for the pod to be running")

	discoverer := discovery.NewDiscoverer(nil, kubelet)
	discoveredItems, err := discoverer.Run()
	require.NoErrorf(t, err, "running discovery with no namespaces")

	var desiredItem *discovery.DiscoveredItem

	for _, item := range discoveredItems {
		if item.MetricAnnotations["podName"] == pod.Name {
			desiredItem = &item
			break
		}
	}

	require.NotNil(t, desiredItem, "expected discovered item not found")

	// Use contains for comparing the image, as containerd runtime will e.g. transform nginx:latest from pod spec
	// to docker.io/library/nginx:latest.
	assert.Contains(t, desiredItem.MetricAnnotations["image"], pod.Spec.Containers[0].Image)

	assert.Equal(t, pod.ObjectMeta.Annotations["key"], desiredItem.MetricAnnotations["label.key"])
	assert.Equal(t, pod.ObjectMeta.Annotations["key2"], desiredItem.MetricAnnotations["label.key2"])
	assert.Equal(t, pod.Spec.Containers[0].Name, desiredItem.MetricAnnotations["name"])
	assert.Equal(t, pod.Namespace, desiredItem.MetricAnnotations["namespace"])

	require.Len(t, desiredItem.EntityRewrites, 1)
	assert.Equal(t, "replace", desiredItem.EntityRewrites[0].Action)
	assert.Equal(t, "${ip}", desiredItem.EntityRewrites[0].Match)
	assert.Equal(t, "k8s:${clusterName}:${namespace}:pod:${podName}:${name}", desiredItem.EntityRewrites[0].ReplaceField)

	assert.Equal(t, pod.ObjectMeta.Annotations["key"], desiredItem.Variables["annotation.key"])
	assert.Contains(t, desiredItem.Variables["image"], pod.Spec.Containers[0].Image)
	assert.Equal(t, pod.ObjectMeta.Labels["key"], desiredItem.Variables["label.key"])
	assert.Equal(t, pod.ObjectMeta.Labels["key2"], desiredItem.Variables["label.key2"])
	assert.Equal(t, pod.Name, desiredItem.Variables["podName"])
	assert.Equal(t, pod.Status.PodIP, desiredItem.Variables["ip"])
	assert.Equal(t, pod.Status.HostIP, desiredItem.Variables["nodeIP"])
}

const (
	// Arbitrary amount of time to let tests exit cleanly before main process terminates.
	timeoutGracePeriod = 10 * time.Second
)

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
