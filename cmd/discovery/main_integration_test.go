// Copyright 2021 New Relic Corporation. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

// +build integration

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	netHTTP "net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/newrelic/nri-discovery-kubernetes/internal/discovery"
	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
)

func Test_Integration(t *testing.T) {
	// Parse kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(utils.HomeDir(), ".kube", "config"))
	if err != nil {
		t.Fatal(err)
	}

	// create the k8s client to populate cluster
	clientset, err := k8s.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	// populate cluster
	pod, err := clientset.CoreV1().Pods("default").Create(getPod())
	if err != nil {
		t.Fatal(err)
	}
	defer clientset.CoreV1().Pods("default").Delete(pod.Name, &metav1.DeleteOptions{})

	// We need the pod running and 7 is a magic number.
	time.Sleep(7 * time.Second)

	// getClientForMinikube.
	e2eClient := getMinikubeHTTPClient(t, *config)

	// getting Kubelet injecting client.
	kubelet, err := kubernetes.NewKubeletWithClient(&e2eClient)
	if err != nil {
		t.Fatal(err)
	}

	// Running discovery.
	discoverer := discovery.NewDiscoverer(nil, kubelet)
	discovered, err := discoverer.Run()
	if err != nil {
		t.Fatal(err)
	}

	var resultToTest discovery.DiscoveredItem
	for _, d := range discovered {
		if d.MetricAnnotations["podName"] == pod.Name {
			resultToTest = d
		}
	}

	assert.Equal(t, "nginx:latest", resultToTest.MetricAnnotations["image"])
	assert.Equal(t, "value", resultToTest.MetricAnnotations["label.key"])
	assert.Equal(t, "value2", resultToTest.MetricAnnotations["label.key2"])
	assert.Equal(t, "test-container", resultToTest.MetricAnnotations["name"])
	assert.Equal(t, "default", resultToTest.MetricAnnotations["namespace"])

	require.Len(t, resultToTest.EntityRewrites, 1)
	assert.Equal(t, "replace", resultToTest.EntityRewrites[0].Action)
	assert.Equal(t, "${ip}", resultToTest.EntityRewrites[0].Match)
	assert.Equal(t, "k8s:${clusterName}:${namespace}:pod:${podName}:${name}", resultToTest.EntityRewrites[0].ReplaceField)

	assert.Equal(t, "value", resultToTest.Variables["annotation.key"])
	assert.Equal(t, "nginx:latest", resultToTest.Variables["image"])
	assert.Equal(t, "value", resultToTest.Variables["label.key"])
	assert.Equal(t, "value2", resultToTest.Variables["label.key2"])
	assert.Equal(t, pod.Name, resultToTest.Variables["podName"])
	assert.Equal(t, kubernetes.PortsMap{"0": 8082, "1": 8084, "port-1": 8082, "port-2": 8084}, resultToTest.Variables["ports"])
}

// In order to connect to Minikube from outside of the cluster we need to setup certificates
func getMinikubeHTTPClient(t *testing.T, clientConfig rest.Config) http.HttpClient {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(filepath.Join(utils.HomeDir(), "/.minikube/profiles/minikube/client.crt"),
		filepath.Join(utils.HomeDir(), "/.minikube/profiles/minikube/client.key"))
	if err != nil {
		t.Fatal(err)
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(filepath.Join(utils.HomeDir(), "/.minikube/ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := netHTTP.Client{
		Transport: &netHTTP.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				Certificates:       []tls.Certificate{cert},
				RootCAs:            caCertPool,
			},
		},
	}

	urlMinikube, err := url.Parse(clientConfig.Host)
	if err != nil {
		t.Fatal(err)
	}

	url, err := url.Parse("https://" + urlMinikube.Hostname() + ":10250")
	if err != nil {
		t.Fatal(err)
	}

	e2eClient := minikubeClient{
		http: client,
		url:  *url,
	}
	return &e2eClient
}

type minikubeClient struct {
	http netHTTP.Client
	url  url.URL
}

func (e *minikubeClient) Get(path string) ([]byte, error) {
	endpoint := e.url.String() + path
	req, _ := netHTTP.NewRequest(netHTTP.MethodGet, endpoint, nil)

	resp, err := e.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	buff, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != netHTTP.StatusOK {
		return nil, fmt.Errorf("httpClient buffer: '%s', Status %s", string(buff), resp.Status)
	}

	return buff, nil
}

func getPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-pod-",
			Labels: map[string]string{
				"key":  "value",
				"key2": "value2",
			},
			Annotations: map[string]string{
				"key": "value",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx",
					Ports: []corev1.ContainerPort{
						{
							Name:          "port-1",
							HostPort:      8081,
							ContainerPort: 8082,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "port-2",
							HostPort:      8083,
							ContainerPort: 8084,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
			},
		},
	}
}
