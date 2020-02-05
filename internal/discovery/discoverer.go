package discovery

import (
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
	"strings"
)

// Replacement defines actions to take to format entity name
type Replacement struct {
	Action       string `json:"action"`
	Match        string `json:"match"`
	ReplaceField string `json:"replaceField"`
}

// DiscoveredItem defines the structure of a single item that has been "discovered"
type DiscoveredItem struct {
	Variables         map[string]interface{} `json:"variables"`
	MetricAnnotations map[string]interface{} `json:"metricAnnotations"`
	EntityRewrites    []Replacement          `json:"entityRewrites"`
}

// Output defines the final output of the discovery executable
type Output []DiscoveredItem

// Discoverer implements the specific discovery mechanism
type Discoverer struct {
	namespaces []string
	kubelet    kubernetes.Kubelet
}

// NewDiscoverer creates a new discoverer implementation
func NewDiscoverer(namespaces []string, kubelet kubernetes.Kubelet) *Discoverer {
	return &Discoverer{
		namespaces: namespaces,
		kubelet:    kubelet,
	}
}

// Run executes the discovery mechanism
func (d *Discoverer) Run() (Output, error) {
	pods, err := d.kubelet.FindContainers(d.namespaces)
	return processContainers(pods), err
}

func processContainers(containers []kubernetes.ContainerInfo) Output {
	//default empty, instead of nil
	output := Output{}
	for _, c := range containers {
		// new map for each container
		var discoveredProperties = make(map[string]interface{})

		discoveredProperties[namespace] = c.Namespace
		discoveredProperties[podName] = c.PodName
		discoveredProperties[ip] = c.PodIP
		discoveredProperties[cluster] = c.Cluster
		discoveredProperties[node] = c.NodeName
		discoveredProperties[nodeIP] = c.NodeIP
		// although labels are set in the pods, we "apply" them to containers
		for k, v := range c.PodLabels {
			discoveredProperties[labelPrefix+k] = v
		}
		discoveredProperties[id] = c.ID
		discoveredProperties[name] = c.Name
		discoveredProperties[image] = c.Image
		discoveredProperties[ports] = c.Ports
		// although annotation are set in the pods, we "apply" them to containers
		for k, v := range c.PodAnnotations {
			discoveredProperties[annotationPrefix+k] = v
		}
		//remove from discovered properties, k8s annotations
		metricAnnotations := filterAnnotations(discoveredProperties)

		item := DiscoveredItem{
			Variables:         discoveredProperties,
			MetricAnnotations: metricAnnotations,
			EntityRewrites:    getReplacements(),
		}
		output = append(output, item)
	}

	return output
}

func getReplacements() []Replacement {
	return []Replacement{
		{
			Action:       entityRewriteActionReplace,
			Match:        entityRewriteMatch,
			ReplaceField: entityReplaceField,
		},
	}
}

var annotationExclusions = []string{
	id, ip, nodeIP, ports,
}

func filterAnnotations(props map[string]interface{}) map[string]interface{} {
	filtered := make(map[string]interface{})
	for k, v := range props {
		if strings.HasPrefix(k, annotationPrefix) {
			continue
		}

		if utils.Contains(annotationExclusions, k) {
			continue
		}

		filtered[k] = v
	}

	return filtered
}
