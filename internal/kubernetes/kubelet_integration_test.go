// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build integration

package kubernetes_test

import (
	"context"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
)

func Test_Kubelet_client_finds_containers_info_using_TLS_and_client_certificate_from_kubeconfig(t *testing.T) {
	t.Parallel()

	cfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(utils.HomeDir(), ".kube", "config"))
	require.NoErrorf(t, err, "building config from kubeconfig")

	clientset, err := k8s.NewForConfig(cfg)
	require.NoErrorf(t, err, "creating k8s client")

	nodes, err := clientset.CoreV1().Nodes().List(contextWithDeadline(t), metav1.ListOptions{})
	require.NoErrorf(t, err, "listing nodes")

	require.Equalf(t, len(nodes.Items), 1, "expected only one Node on the cluster")

	clusterURL, err := url.Parse(cfg.Host)
	require.NoErrorf(t, err, "parsing API URL")

	kubeletHost := clusterURL.Hostname()
	kubeletPort := 10250
	useTLS := true
	autoConfig := false
	discoveryTimeout := 5 * time.Minute

	kubelet, err := kubernetes.NewKubelet(kubeletHost, kubeletPort, useTLS, autoConfig, discoveryTimeout)
	require.NoErrorf(t, err, "creating kubelet client")

	_, err = kubelet.FindContainers(nil)
	require.NoErrorf(t, err, "finding containers")
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
