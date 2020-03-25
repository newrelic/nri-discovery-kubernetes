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
const kubeletHostEnvVar = "NRK8S_NODE_NAME"

type PortsMap map[string]int32
type LabelsMap map[string]string
type AnnotationsMap map[string]string

type ContainerInfo struct {
	Name           string
	ID             string
	Image          string
	ImageID        string
	Ports          PortsMap
	PodLabels      LabelsMap
	PodAnnotations AnnotationsMap
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
		for idx, cs := range pod.Status.ContainerStatuses {
			if cs.State.Running != nil || cs.State.Waiting != nil {
				ports := getPorts(pod, idx)
				c := ContainerInfo{
					Name:           cs.Name,
					ID:             cs.ContainerID,
					Image:          cs.Image,
					ImageID:        cs.ImageID,
					Ports:          ports,
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

func getPorts(pod v1.Pod, containerIndex int) PortsMap {
	ports := make(PortsMap)
	if len(pod.Spec.Containers) > 0 &&
		len(pod.Spec.Containers[containerIndex].Ports) > 0 {
		// we add the port index and if available the name.
		// you can then use either to refer to the value
		for portIndex, port := range pod.Spec.Containers[containerIndex].Ports {
			ports[strconv.Itoa(portIndex)] = port.ContainerPort
			if len(port.Name) > 0 {
				ports[port.Name] = port.ContainerPort
			}
		}
	}
	return ports
}

func getClusterName() string {
	// there is no way at the moment to get the cluster name from the Kubelet API
	clusterName := os.Getenv(clusterNameEnvVar)
	return clusterName
}

func NewKubelet(port int, useTLS bool) (Kubelet, error) {
	config, err := rest.InClusterConfig()
	// not inside the cluster?
	if err != nil {
		kConfigPath := filepath.Join(utils.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster configuration: %s", err)
		}
	}

	clusterName := getClusterName()
	kubeletHost, isKubeletHostSet := os.LookupEnv(kubeletHostEnvVar)

	if !isKubeletHostSet {
		// If the environment variable represented by kubeletHostEnvVar is not set,
		// fallback to the default value.
		kubeletHost = host
	}

	hostUrl := makeUrl(kubeletHost, port, useTLS)
	httpClient := http.NewClient(hostUrl, config.BearerToken)

	kubelet := &kubelet{
		client:      &httpClient,
		NodeName:    kubeletHost,
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

func makeUrl(host string, port int, useTLS bool) url.URL {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	kubeletUrl := url.URL{
		Scheme: scheme,
		Host:   host + ":" + strconv.Itoa(port),
	}
	return kubeletUrl
}
