package kubernetes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	podsPath          = "/pods"
	clusterNameEnvVar = "CLUSTER_NAME"
	nodeNameEnvVar    = "NRK8S_NODE_NAME"
)

type (
	PortsMap       map[string]int32
	LabelsMap      map[string]string
	AnnotationsMap map[string]string
)

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

func (kube *kubelet) getPods() ([]corev1.Pod, error) {
	resp, err := (*kube.client).Get(podsPath)
	if err != nil {
		err = fmt.Errorf("failed to execute request against kubelet: %s ", err)
		return []corev1.Pod{}, err
	}

	pods := &corev1.PodList{}
	err = json.Unmarshal(resp, pods)
	if err != nil {
		err = fmt.Errorf("failed to unmarshall result in to list of pods: %s", err)
	}
	return pods.Items, err
}

func filterByNamespace(allPods []corev1.Pod, namespaces []string) []corev1.Pod {
	if len(namespaces) == 0 {
		return allPods
	}

	var result []corev1.Pod
	for _, pod := range allPods {
		if utils.Contains(namespaces, pod.Namespace) {
			result = append(result, pod)
		}
	}
	return result
}

func getContainers(clusterName string, nodeName string, pods []corev1.Pod) []ContainerInfo {
	var containers []ContainerInfo

	for _, pod := range pods {

		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		for idx, cs := range pod.Status.ContainerStatuses {

			if cs.State.Running == nil {
				continue
			}

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
	return containers
}

func getPorts(pod corev1.Pod, containerIndex int) PortsMap {
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

func NewKubelet(host string, port int, useTLS bool, autoConfig bool, timeout time.Duration) (Kubelet, error) {
	restConfig, err := rest.InClusterConfig()
	// not inside the cluster?
	if err != nil {
		kConfigPath := filepath.Join(utils.HomeDir(), ".kube", "config")
		restConfig, err = clientcmd.BuildConfigFromFlags("", kConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster configuration: %s", err)
		}
	}

	clusterName := getClusterName()
	nodeName, isNodeNameSet := os.LookupEnv(nodeNameEnvVar)
	if autoConfig && isNodeNameSet {
		client, err := http.NewKubeletClient(nodeName, timeout)
		if err != nil {
			logrus.WithError(err).Warn("failed to initialize kubelet client")
		} else {
			kubelet := &kubelet{
				client:      &client,
				NodeName:    nodeName,
				ClusterName: clusterName,
			}
			return kubelet, nil
		}
	}

	// host provided by cmd line arg has higher precedence.
	// if host cmd line arg is not provided use NRK8S_NODE_NAME in case is set, otherwise localhost.
	kubeletHost := host
	if isNodeNameSet && !config.IsFlagPassed(config.Host) {
		kubeletHost = nodeName
	}

	hostUrl := makeUrl(kubeletHost, port, useTLS)
	httpClient := http.NewClient(hostUrl, restConfig.BearerToken)

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
