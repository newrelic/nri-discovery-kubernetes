package kubelet

import (
	"encoding/json"
	"fmt"
	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
	"github.com/sirupsen/logrus"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path"
	"strconv"
)

const (
	podsPath = "/pods"
)

type (
	// PortsMap stores container ports indexed by name.
	PortsMap map[string]int32
	// LabelsMap stores Pod labels.
	LabelsMap map[string]string
	// AnnotationsMap stores Pod annotations.
	AnnotationsMap map[string]string
)

// ContainerInfo represents discovery-specific format for found Pods via Kubelet API.
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

// Kubelet defines what functionality kubelet client provides.
type Kubelet interface {
	FindContainers(namespaces []string) ([]ContainerInfo, error)
}

type kubelet struct {
	client      *http.Client
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
	resp, err := kube.client.Get(podsPath)
	if err != nil {
		err = fmt.Errorf("failed to execute request against kubelet: %s ", err)
		return []corev1.Pod{}, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("failed read the response from kubelet: %s ", err)
		return []corev1.Pod{}, err
	}

	pods := &corev1.PodList{}
	err = json.Unmarshal(respBody, pods)
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

// New validates and constructs Kubelet client.
func New(config *config.Config) (Kubelet, error) {
	k8sConfig, err := getK8sConfig(config)
	if err != nil {
		return nil, fmt.Errorf("setting kubernetes configuration: %w", err)
	}

	k8s, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("building kubernetes client: %w", err)
	}

	connector := http.DefaultConnector(k8s, config, k8sConfig)

	kubeletClient, err := http.New(connector, http.WithMaxRetries(5)) // TODO: Retries are hardcoded
	if err != nil {
		return nil, fmt.Errorf("building kubelet client: %w", err)
	}

	k := kubelet{
		client:      kubeletClient,
		ClusterName: config.ClusterName,
		NodeName:    config.NodeName,
	}

	return &k, nil
}

func getK8sConfig(c *config.Config) (*rest.Config, error) {
	inclusterConfig, err := rest.InClusterConfig()
	if err == nil {
		return inclusterConfig, nil
	}
	logrus.Warnf("collecting in cluster config: %v", err)

	kubeconf := c.KubeConfigFile
	if kubeconf == "" {
		kubeconf = path.Join(homedir.HomeDir(), ".kube", "config")
	}

	inclusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeconf)
	if err != nil {
		return nil, fmt.Errorf("could not load local kube config: %w", err)
	}

	logrus.Warnf("using local kube config: %q", kubeconf)

	return inclusterConfig, nil
}
