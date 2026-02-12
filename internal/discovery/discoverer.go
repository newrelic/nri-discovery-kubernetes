package discovery

import (
	"fmt"
	"strings"

	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/newrelic/nri-discovery-kubernetes/internal/utils"
)

// Replacement defines actions to take to format entity name.
type Replacement struct {
	Action       string `json:"action"`
	Match        string `json:"match"`
	ReplaceField string `json:"replaceField"`
}

type (
	// VariablesMap is used to store discovery properties.
	VariablesMap map[string]interface{}
	// AnnotationsMap is used to discovered annotations.
	AnnotationsMap = VariablesMap
)

// DiscoveredItem defines the structure of a single item that has been "discovered".
type DiscoveredItem struct {
	Variables         VariablesMap   `json:"variables"`
	MetricAnnotations AnnotationsMap `json:"metricAnnotations"`
	EntityRewrites    []Replacement  `json:"entityRewrites"`
}

// Output defines the final output of the discovery executable.
type Output []DiscoveredItem

// Discoverer implements the specific discovery mechanism.
type Discoverer struct {
	namespaces        []string
	kubelet           kubernetes.Kubelet
	serviceDiscoverer kubernetes.ServiceDiscoverer
	discoverServices  bool
}

// NewDiscoverer creates a new discoverer implementation (containers only by default).
func NewDiscoverer(namespaces []string, kubelet kubernetes.Kubelet, discoverServices bool) *Discoverer {
	return &Discoverer{
		namespaces:       namespaces,
		kubelet:          kubelet,
		discoverServices: discoverServices,
	}
}

// SetServiceDiscoverer sets the service discoverer for discovering services.
func (d *Discoverer) SetServiceDiscoverer(sd kubernetes.ServiceDiscoverer) {
	d.serviceDiscoverer = sd
}

// Run executes the discovery mechanism.
func (d *Discoverer) Run() (Output, error) {
	output := Output{}

	if !d.discoverServices {
		// Default: discover containers only
		pods, err := d.kubelet.FindContainers(d.namespaces)
		if err != nil {
			return nil, err
		}
		output = append(output, processContainers(pods)...)
	} else {
		// Discover services instead of containers
		if d.serviceDiscoverer == nil {
			return nil, fmt.Errorf("service discoverer not configured but discover-services flag is set")
		}
		services, err := d.serviceDiscoverer.FindServices(d.namespaces)
		if err != nil {
			return nil, err
		}
		output = append(output, processServices(services)...)
	}

	return output, nil
}

func processContainers(containers []kubernetes.ContainerInfo) Output {
	// default empty, instead of nil.
	output := Output{}
	for _, c := range containers {
		// new map for each container.
		discoveredProperties := make(VariablesMap)

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
		// remove from discovered properties, k8s annotations
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

func filterAnnotations(props VariablesMap) AnnotationsMap {
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

func processServices(services []kubernetes.ServiceInfo) Output {
	// default empty, instead of nil.
	output := Output{}
	for _, svc := range services {
		// new map for each service.
		discoveredProperties := make(VariablesMap)

		discoveredProperties[cluster] = svc.Cluster
		discoveredProperties[namespace] = svc.Namespace
		discoveredProperties[serviceName] = svc.Name
		discoveredProperties[serviceType] = svc.Type
		discoveredProperties[clusterIP] = svc.ClusterIP
		if len(svc.ExternalIPs) > 0 {
			discoveredProperties[externalIPs] = svc.ExternalIPs
		}
		discoveredProperties[ports] = svc.Ports

		// Add service selector labels
		for k, v := range svc.Selector {
			discoveredProperties[labelPrefix+k] = v
		}

		// Add service labels
		for k, v := range svc.Labels {
			discoveredProperties[labelPrefix+k] = v
		}

		// Add service annotations
		for k, v := range svc.Annotations {
			discoveredProperties[annotationPrefix+k] = v
		}

		// remove from discovered properties, k8s annotations
		metricAnnotations := filterAnnotations(discoveredProperties)

		item := DiscoveredItem{
			Variables:         discoveredProperties,
			MetricAnnotations: metricAnnotations,
			EntityRewrites: []Replacement{
				{
					Action:       entityRewriteActionReplace,
					Match:        "${" + clusterIP + "}",
					ReplaceField: serviceEntityReplaceField,
				},
			},
		}
		output = append(output, item)
	}

	return output
}
