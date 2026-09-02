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
	Variables           VariablesMap   `json:"variables"`
	MetricAnnotations   AnnotationsMap `json:"metricAnnotations"`
	EntityRewrites      []Replacement  `json:"entityRewrites"`
	Instrumented        bool           `json:"instrumented"`
	InstrumentationType string         `json:"instrumentationType,omitempty"`
}

// Output defines the final output of the discovery executable.
type Output []DiscoveredItem

// Discoverer implements the specific discovery mechanism.
type Discoverer struct {
	namespaces        []string
	kubelet           kubernetes.Kubelet
	serviceDiscoverer kubernetes.ServiceDiscoverer
	workloadDiscoverer kubernetes.WorkloadDiscoverer
	pvcDiscoverer     kubernetes.PVCDiscoverer
	crdDiscoverer     kubernetes.CRDDiscoverer
	discoverServices  bool
	// resources is the explicit list of resource types to discover.
	// When non-empty it drives Run() instead of discoverServices.
	resources []string
}

// NewDiscoverer creates a new discoverer implementation (containers only by default).
func NewDiscoverer(namespaces []string, kubelet kubernetes.Kubelet, discoverServices bool) *Discoverer {
	return &Discoverer{
		namespaces:       namespaces,
		kubelet:          kubelet,
		discoverServices: discoverServices,
	}
}

// NewMultiDiscoverer creates a discoverer for multiple resource types at once.
func NewMultiDiscoverer(namespaces []string, resources []string) *Discoverer {
	return &Discoverer{
		namespaces: namespaces,
		resources:  resources,
	}
}

// SetKubelet sets the kubelet client (needed when "pods" is in the resources list).
func (d *Discoverer) SetKubelet(k kubernetes.Kubelet) {
	d.kubelet = k
}

// SetServiceDiscoverer sets the service discoverer for discovering services.
func (d *Discoverer) SetServiceDiscoverer(sd kubernetes.ServiceDiscoverer) {
	d.serviceDiscoverer = sd
}

// SetWorkloadDiscoverer sets the workload discoverer.
func (d *Discoverer) SetWorkloadDiscoverer(wd kubernetes.WorkloadDiscoverer) {
	d.workloadDiscoverer = wd
}

// SetPVCDiscoverer sets the PVC discoverer.
func (d *Discoverer) SetPVCDiscoverer(pd kubernetes.PVCDiscoverer) {
	d.pvcDiscoverer = pd
}

// SetCRDDiscoverer sets the CRD discoverer.
func (d *Discoverer) SetCRDDiscoverer(cd kubernetes.CRDDiscoverer) {
	d.crdDiscoverer = cd
}

// Run executes the discovery mechanism.
func (d *Discoverer) Run() (Output, error) {
	// Multi-resource mode: --resources flag was specified
	if len(d.resources) > 0 {
		return d.runMulti()
	}

	// Legacy single-mode: --discover-services or pod/container default
	output := Output{}
	if !d.discoverServices {
		pods, err := d.kubelet.FindContainers(d.namespaces)
		if err != nil {
			return nil, err
		}
		output = append(output, processContainers(pods)...)
	} else {
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

func (d *Discoverer) runMulti() (Output, error) {
	output := Output{}
	seen := make(map[string]bool)

	for _, res := range d.resources {
		if seen[res] {
			continue
		}
		seen[res] = true

		switch res {
		case "pods":
			if d.kubelet == nil {
				continue
			}
			pods, err := d.kubelet.FindContainers(d.namespaces)
			if err != nil {
				return nil, fmt.Errorf("discovering pods: %w", err)
			}
			output = append(output, processContainers(pods)...)

		case "services":
			if d.serviceDiscoverer == nil {
				continue
			}
			svcs, err := d.serviceDiscoverer.FindServices(d.namespaces)
			if err != nil {
				return nil, fmt.Errorf("discovering services: %w", err)
			}
			output = append(output, processServices(svcs)...)

		case "deployments", "statefulsets", "daemonsets":
			// All three share one discoverer — run it once regardless of which
			// subset appeared in the resources list.
			if seen["_workloads"] || d.workloadDiscoverer == nil {
				continue
			}
			seen["_workloads"] = true
			workloads, err := d.workloadDiscoverer.FindWorkloads(d.namespaces)
			if err != nil {
				return nil, fmt.Errorf("discovering workloads: %w", err)
			}
			output = append(output, processWorkloads(workloads)...)

		case "pvcs":
			if d.pvcDiscoverer == nil {
				continue
			}
			pvcs, err := d.pvcDiscoverer.FindPVCs(d.namespaces)
			if err != nil {
				return nil, fmt.Errorf("discovering PVCs: %w", err)
			}
			output = append(output, processPVCs(pvcs)...)

		case "crds":
			if d.crdDiscoverer == nil {
				continue
			}
			crds, err := d.crdDiscoverer.FindCRDs()
			if err != nil {
				return nil, fmt.Errorf("discovering CRDs: %w", err)
			}
			output = append(output, processCRDs(crds)...)
		}
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
		discoveredProperties[kind] = "Pod"
		discoveredProperties[id] = c.ID
		discoveredProperties[name] = c.Name
		discoveredProperties[image] = c.Image
		discoveredProperties[ports] = c.Ports
		discoveredProperties[instrumented] = c.Instrumented
		if c.InstrumentationType != "" {
			discoveredProperties[instrumentationType] = c.InstrumentationType
		}
		// although annotation are set in the pods, we "apply" them to containers
		for k, v := range c.PodAnnotations {
			discoveredProperties[annotationPrefix+k] = v
		}
		// remove from discovered properties, k8s annotations
		metricAnnotations := filterAnnotations(discoveredProperties)

		item := DiscoveredItem{
			Variables:           discoveredProperties,
			MetricAnnotations:   metricAnnotations,
			EntityRewrites:      getReplacements(),
			Instrumented:        c.Instrumented,
			InstrumentationType: c.InstrumentationType,
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

func processWorkloads(workloads []kubernetes.WorkloadInfo) Output {
	output := Output{}
	for _, w := range workloads {
		props := make(VariablesMap)
		props[kind]                      = string(w.Kind)
		props[cluster]                   = w.Cluster
		props[namespace]                 = w.Namespace
		props[workloadName]              = w.Name
		props[workloadReplicas]          = w.Replicas
		props[workloadReadyReplicas]     = w.ReadyReplicas
		props[workloadAvailableReplicas] = w.AvailableReplicas
		props[workloadImages]            = strings.Join(w.Images, ",")
		props[instrumented]              = w.Instrumented
		if w.InstrumentationType != "" {
			props[instrumentationType] = w.InstrumentationType
		}
		for k, v := range w.Labels {
			props[labelPrefix+k] = v
		}
		for k, v := range w.Annotations {
			props[annotationPrefix+k] = v
		}

		metricAnns := filterAnnotations(props)
		output = append(output, DiscoveredItem{
			Variables:           props,
			MetricAnnotations:   metricAnns,
			EntityRewrites:      []Replacement{{Action: entityRewriteActionReplace, Match: "${clusterIP}", ReplaceField: workloadEntityReplaceField}},
			Instrumented:        w.Instrumented,
			InstrumentationType: w.InstrumentationType,
		})
	}
	return output
}

func processPVCs(pvcs []kubernetes.PVCInfo) Output {
	output := Output{}
	for _, p := range pvcs {
		props := make(VariablesMap)
		props[kind]            = "PersistentVolumeClaim"
		props[cluster]         = p.Cluster
		props[namespace]       = p.Namespace
		props[pvcName]         = p.Name
		props[pvcStorageClass] = p.StorageClass
		props[pvcAccessModes]  = accessModesString(p.AccessModes)
		props[pvcCapacity]     = p.Capacity
		props[pvcStatus]       = p.Status
		props[pvcVolumeName]   = p.VolumeName
		for k, v := range p.Labels {
			props[labelPrefix+k] = v
		}
		for k, v := range p.Annotations {
			props[annotationPrefix+k] = v
		}

		metricAnns := filterAnnotations(props)
		output = append(output, DiscoveredItem{
			Variables:         props,
			MetricAnnotations: metricAnns,
			EntityRewrites:    []Replacement{{Action: entityRewriteActionReplace, Match: "${pvcName}", ReplaceField: pvcEntityReplaceField}},
		})
	}
	return output
}

func processCRDs(crds []kubernetes.CRDInfo) Output {
	output := Output{}
	for _, c := range crds {
		props := make(VariablesMap)
		props[kind]        = "CustomResourceDefinition"
		props[crdName]     = c.Name
		props[crdGroup]    = c.Group
		props[crdScope]    = c.Scope
		props[crdVersion]  = c.Version
		props[instrumented] = false
		for k, v := range c.Labels {
			props[labelPrefix+k] = v
		}
		for k, v := range c.Annotations {
			props[annotationPrefix+k] = v
		}

		metricAnns := filterAnnotations(props)
		output = append(output, DiscoveredItem{
			Variables:         props,
			MetricAnnotations: metricAnns,
			EntityRewrites:    []Replacement{{Action: entityRewriteActionReplace, Match: "${crdName}", ReplaceField: "k8s:crd:${crdGroup}:${crdName}"}},
		})
	}
	return output
}

func accessModesString(modes []string) string {
	return strings.Join(modes, ",")
}

func processServices(services []kubernetes.ServiceInfo) Output {
	// default empty, instead of nil.
	output := Output{}
	for _, svc := range services {
		// new map for each service.
		discoveredProperties := make(VariablesMap)

		discoveredProperties[kind] = "Service"
		discoveredProperties[cluster] = svc.Cluster
		discoveredProperties[namespace] = svc.Namespace
		discoveredProperties[serviceName] = svc.Name
		discoveredProperties[serviceType] = svc.Type
		discoveredProperties[clusterIP] = svc.ClusterIP
		if len(svc.ExternalIPs) > 0 {
			discoveredProperties[externalIPs] = svc.ExternalIPs
		}
		discoveredProperties[ports] = svc.Ports
		discoveredProperties[instrumented] = svc.Instrumented
		if svc.InstrumentationType != "" {
			discoveredProperties[instrumentationType] = svc.InstrumentationType
		}

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
			Variables:           discoveredProperties,
			MetricAnnotations:   metricAnnotations,
			EntityRewrites: []Replacement{
				{
					Action:       entityRewriteActionReplace,
					Match:        "${" + clusterIP + "}",
					ReplaceField: serviceEntityReplaceField,
				},
			},
			Instrumented:        svc.Instrumented,
			InstrumentationType: svc.InstrumentationType,
		}
		output = append(output, item)
	}

	return output
}
