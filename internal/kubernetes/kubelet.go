package kubernetes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const host = "localhost"
const podsPath = "/pods"
const clusterNameEnvVar = "CLUSTER_NAME"

type ContainerInfo struct {
	Name           string
	ID             string
	Image          string
	ImageID        string
	PodLabels      map[string]string
	PodAnnotations map[string]string
	PodIP          string
	PodName        string
	NodeName       string
	NodeIP         string
	Namespace      string
	Cluster        string
}

type Kubelet interface {
	FindContainers(namespaces []string) ([]ContainerInfo, error)
}

type kubelet struct {
	client      *http.HttpClient
	NodeName    string
	ClusterName string
}

func (kube *kubelet) FindContainers(namespaces []string) ([]ContainerInfo, error) {
	allPods, err := kube.getPods()
	if err != nil {
		return nil, err
	}
	pods := filterByNamespace(allPods, namespaces)
	return getContainers(kube.ClusterName, kube.NodeName, pods), nil
}

func (kube *kubelet) getPods() ([]v1.Pod, error) {
	resp, err := (*kube.client).Get(podsPath)
	if err != nil {
		err = fmt.Errorf("failed to execute request against kubelet: %s ", err)
		return []v1.Pod{}, err
	}

	var pods = &v1.PodList{}
	err = json.Unmarshal(resp, pods)
	if err != nil {
		err = fmt.Errorf("failed to unmarshall result in to list of pods: %s", err)
	}
	return pods.Items, err
}

func filterByNamespace(allPods []v1.Pod, namespaces []string) []v1.Pod {
	if len(namespaces) == 0 {
		return allPods
	}

	var result []v1.Pod
	for _, pod := range allPods {
		if utils.Contains(namespaces, pod.Namespace) {
			result = append(result, pod)
		}
	}
	return result
}

func getContainers(clusterName string, nodeName string, pods []v1.Pod) []ContainerInfo {
	var containers []ContainerInfo

	for _, pod := range pods {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Running != nil || cs.State.Waiting != nil {
				c := ContainerInfo{
					Name:           cs.Name,
					ID:             cs.ContainerID,
					Image:          cs.Image,
					ImageID:        cs.ImageID,
					PodIP:          pod.Status.PodIP,
					PodLabels:      pod.Labels,
					PodAnnotations: pod.Annotations,
					PodName:        pod.Name,
					NodeName:       nodeName,
					NodeIP:         pod.Status.HostIP,
					Namespace:      pod.Namespace,
					Cluster:        clusterName,
				}
				containers = append(containers, c)
			}
		}
	}
	return containers
}

func getNodeAndClusterName() (string, string) {
	// per docs, hostname should be the node name
	// https://kubernetes.io/docs/concepts/containers/container-environment-variables/#container-information
	nodeName := utils.Hostname()
	// there is no way at the moment to get the cluster from the API
	clusterName := os.Getenv(clusterNameEnvVar)
	return clusterName, nodeName
}

func NewKubelet(port int, insecure bool) (Kubelet, error) {
	config, err := rest.InClusterConfig()
	// not inside the cluster?
	if err != nil {
		kConfigPath := filepath.Join(utils.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster configuration: %s", err)
		}
	}

	clusterName, nodeName := getNodeAndClusterName()
	hostUrl := makeUrl(port, insecure)
	httpClient := http.NewClient(hostUrl, config.BearerToken)

	kubelet := &kubelet{
		client:      &httpClient,
		NodeName:    nodeName,
		ClusterName: clusterName,
	}

	return kubelet, nil
}

func NewKubeletWithClient(httpClient *http.HttpClient) (Kubelet, error) {
	k := &kubelet{
		client: httpClient,
	}

	return k, nil
}

func makeUrl(port int, insecure bool) url.URL {
	scheme := "https"
	if insecure {
		scheme = "http"
	}
	kubeletUrl := url.URL{
		Scheme: scheme,
		Host:   host + ":" + strconv.Itoa(port),
	}
	return kubeletUrl
}
