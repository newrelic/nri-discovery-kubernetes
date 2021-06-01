// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build integration

package main_test

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	retry "github.com/avast/retry-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/newrelic/nri-discovery-kubernetes/internal/discovery"
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
)

func Test_discovery_when(t *testing.T) {
	cfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(utils.HomeDir(), ".kube", "config"))
	require.NoErrorf(t, err, "not expecting error when building config from kubeconfig")

	k8sClient, err := k8s.NewForConfig(cfg)
	require.NoErrorf(t, err, "not expecting error creating k8s client")

	clusterURL, err := url.Parse(cfg.Host)

	require.NoErrorf(t, err, "not expecting error when waiting for the pod to be running")

	kubelet, err := kubernetes.NewKubelet(clusterURL.Hostname(), 10250, true, false, time.Minute*5)
	require.NoErrorf(t, err, "not expecting error when creating kubelet client")

	ctx := contextWithDeadline(t)

	t.Run("succeeds_with_no_namespace", func(t *testing.T) {
		t.Parallel()

		ns := createTestNamespace(t, k8sClient)
		pod := createRunningPod(t, k8sClient, ns.Name)

		discoverer := discovery.NewDiscoverer(nil, kubelet)
		discovered, err := discoverer.Run()
		require.NoErrorf(t, err, "not expecting error when running discovery with no namespaces")

		var resultToTest discovery.DiscoveredItem
		for _, d := range discovered {
			if d.MetricAnnotations["podName"] == pod.Name {
				resultToTest = d
			}
		}
		require.NotEqualf(t, resultToTest, discovery.DiscoveredItem{}, "we expect resultToTest to be populated")
		testPodIntegration(t, resultToTest, pod)
	})

	t.Run("succeeds_with_valid_namespace", func(t *testing.T) {
		t.Parallel()

		ns := createTestNamespace(t, k8sClient)
		pod := createRunningPod(t, k8sClient, ns.Name)

		discoverer := discovery.NewDiscoverer([]string{pod.Namespace}, kubelet)
		discovered, err := discoverer.Run()
		require.NoErrorf(t, err, "not expecting error when running discovery with valid namespaces")

		var resultToTest discovery.DiscoveredItem
		for _, d := range discovered {
			if d.MetricAnnotations["podName"] == pod.Name {
				resultToTest = d
			}
		}
		require.Len(t, discovered, 1, "we are expecting only one pod")
		require.NotEqualf(t, resultToTest, discovery.DiscoveredItem{}, "we expect resultToTest to be populated")
		testPodIntegration(t, resultToTest, pod)
	})

	t.Run("succeeds_with_not_existing_namespace", func(t *testing.T) {
		t.Parallel()

		ns := createTestNamespace(t, k8sClient)
		createRunningPod(t, k8sClient, ns.Name)

		discoverer := discovery.NewDiscoverer([]string{"not-existing"}, kubelet)
		discovered, err := discoverer.Run()
		require.NoErrorf(t, err, "not expecting error when running discovery with not-existing namespaces")

		require.Len(t, discovered, 0, "we are not expecting any element")
	})

	t.Run("succeeds_with_failing_pod", func(t *testing.T) {
		t.Parallel()

		ns := createTestNamespace(t, k8sClient)

		pod := getPodIntegration()
		pod.Spec.Containers[0].Image = "not-existing-image"
		pod, err := k8sClient.CoreV1().Pods(ns.Name).Create(ctx, pod, metav1.CreateOptions{})
		require.NoErrorf(t, err, "not expecting error when creating test pod")

		discoverer := discovery.NewDiscoverer([]string{"not-existing"}, kubelet)
		discovered, err := discoverer.Run()
		require.NoErrorf(t, err, "not expecting error when running discovery with failing pods")

		require.Len(t, discovered, 0, "we are not expecting any element, since failing pods should be ignored")
	})
}

func createRunningPod(t *testing.T, k8sClient *k8s.Clientset, namespace string) *corev1.Pod {
	ctx := contextWithDeadline(t)

	pod, err := k8sClient.CoreV1().Pods(namespace).Create(ctx, getPodIntegration(), metav1.CreateOptions{})
	require.NoErrorf(t, err, "not expecting error when creating test pod")

	err = retry.Do(func() error {
		pod, err = k8sClient.CoreV1().Pods(namespace).Get(ctx, pod.Name, metav1.GetOptions{})
		require.NoErrorf(t, err, "not expecting error getting pod")
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("the pod is still not running")
		}
		return nil
	},
	)
	require.NoErrorf(t, err, "not expecting error when waiting for the pod to be running")

	return pod
}

func createTestNamespace(t *testing.T, k8sClient *k8s.Clientset) *corev1.Namespace {
	ctx := contextWithDeadline(t)

	ns, err := k8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
		},
	}, metav1.CreateOptions{})
	require.NoErrorf(t, err, "not expecting error when creating test namespace")

	t.Cleanup(func() {
		err := k8sClient.CoreV1().Namespaces().Delete(ctx, ns.Name, metav1.DeleteOptions{})
		require.NoErrorf(t, err, "not expecting error when cleaning environment")
	},
	)

	return ns
}

func getPodIntegration() *corev1.Pod {
	return &corev1.Pod{
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
					Name:  "test-container",
					Image: "nginx:latest",
				},
			},
		},
	}
}

func testPodIntegration(t *testing.T, resultToTest discovery.DiscoveredItem, pod *corev1.Pod) {
	assert.Equal(t, pod.Spec.Containers[0].Image, resultToTest.MetricAnnotations["image"])
	assert.Equal(t, pod.ObjectMeta.Annotations["key"], resultToTest.MetricAnnotations["label.key"])
	assert.Equal(t, pod.ObjectMeta.Annotations["key2"], resultToTest.MetricAnnotations["label.key2"])
	assert.Equal(t, pod.Spec.Containers[0].Name, resultToTest.MetricAnnotations["name"])
	assert.Equal(t, pod.Namespace, resultToTest.MetricAnnotations["namespace"])

	require.Len(t, resultToTest.EntityRewrites, 1)
	assert.Equal(t, "replace", resultToTest.EntityRewrites[0].Action)
	assert.Equal(t, "${ip}", resultToTest.EntityRewrites[0].Match)
	assert.Equal(t, "k8s:${clusterName}:${namespace}:pod:${podName}:${name}", resultToTest.EntityRewrites[0].ReplaceField)

	assert.Equal(t, pod.ObjectMeta.Annotations["key"], resultToTest.Variables["annotation.key"])
	assert.Equal(t, pod.Spec.Containers[0].Image, resultToTest.Variables["image"])
	assert.Equal(t, pod.ObjectMeta.Labels["key"], resultToTest.Variables["label.key"])
	assert.Equal(t, pod.ObjectMeta.Labels["key2"], resultToTest.Variables["label.key2"])
	assert.Equal(t, pod.Name, resultToTest.Variables["podName"])
	assert.Equal(t, pod.Status.PodIP, resultToTest.Variables["ip"])
	assert.Equal(t, pod.Status.HostIP, resultToTest.Variables["nodeIP"])
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
