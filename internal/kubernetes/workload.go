package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// WorkloadKind identifies the Kubernetes workload resource type.
type WorkloadKind string

const (
	KindDeployment  WorkloadKind = "Deployment"
	KindStatefulSet WorkloadKind = "StatefulSet"
	KindDaemonSet   WorkloadKind = "DaemonSet"
)

// WorkloadInfo holds the discovery-specific representation of a workload.
type WorkloadInfo struct {
	Kind                WorkloadKind
	Name                string
	Namespace           string
	Cluster             string
	Labels              LabelsMap
	Annotations         AnnotationsMap
	Images              []string // all container images in the pod template
	Replicas            int32
	ReadyReplicas       int32
	AvailableReplicas   int32
	Instrumented        bool
	InstrumentationType string
}

// WorkloadDiscoverer discovers Deployments, StatefulSets, and DaemonSets.
type WorkloadDiscoverer interface {
	FindWorkloads(namespaces []string) ([]WorkloadInfo, error)
}

type workloadDiscoverer struct {
	clientset   *kubernetes.Clientset
	clusterName string
	// which kinds to include
	deployments  bool
	statefulSets bool
	daemonSets   bool
}

// NewWorkloadDiscoverer creates a discoverer for the given workload kinds.
func NewWorkloadDiscoverer(
	clientset *kubernetes.Clientset,
	cfg *config.Config,
	deployments, statefulSets, daemonSets bool,
) WorkloadDiscoverer {
	return &workloadDiscoverer{
		clientset:    clientset,
		clusterName:  cfg.ClusterName,
		deployments:  deployments,
		statefulSets: statefulSets,
		daemonSets:   daemonSets,
	}
}

func (wd *workloadDiscoverer) FindWorkloads(namespaces []string) ([]WorkloadInfo, error) {
	var result []WorkloadInfo

	namespacesToQuery := namespaces
	if len(namespacesToQuery) == 0 {
		namespacesToQuery = []string{metav1.NamespaceAll}
	}

	for _, ns := range namespacesToQuery {
		if wd.deployments {
			items, err := wd.findDeployments(ns)
			if err != nil {
				return nil, err
			}
			result = append(result, items...)
		}
		if wd.statefulSets {
			items, err := wd.findStatefulSets(ns)
			if err != nil {
				return nil, err
			}
			result = append(result, items...)
		}
		if wd.daemonSets {
			items, err := wd.findDaemonSets(ns)
			if err != nil {
				return nil, err
			}
			result = append(result, items...)
		}
	}

	return result, nil
}

func (wd *workloadDiscoverer) findDeployments(ns string) ([]WorkloadInfo, error) {
	ctx := context.Background()
	list, err := wd.clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing deployments in %q: %w", ns, err)
	}

	result := make([]WorkloadInfo, 0, len(list.Items))
	for _, d := range list.Items {
		instr, instrType := detectTemplateInstrumentation(d.Spec.Template)
		result = append(result, WorkloadInfo{
			Kind:                KindDeployment,
			Name:                d.Name,
			Namespace:           d.Namespace,
			Cluster:             wd.clusterName,
			Labels:              d.Labels,
			Annotations:         d.Annotations,
			Images:              containerImages(d.Spec.Template.Spec),
			Replicas:            derefInt32(d.Spec.Replicas),
			ReadyReplicas:       d.Status.ReadyReplicas,
			AvailableReplicas:   d.Status.AvailableReplicas,
			Instrumented:        instr,
			InstrumentationType: instrType,
		})
	}
	return result, nil
}

func (wd *workloadDiscoverer) findStatefulSets(ns string) ([]WorkloadInfo, error) {
	ctx := context.Background()
	list, err := wd.clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing statefulsets in %q: %w", ns, err)
	}

	result := make([]WorkloadInfo, 0, len(list.Items))
	for _, s := range list.Items {
		instr, instrType := detectTemplateInstrumentation(s.Spec.Template)
		result = append(result, WorkloadInfo{
			Kind:                KindStatefulSet,
			Name:                s.Name,
			Namespace:           s.Namespace,
			Cluster:             wd.clusterName,
			Labels:              s.Labels,
			Annotations:         s.Annotations,
			Images:              containerImages(s.Spec.Template.Spec),
			Replicas:            derefInt32(s.Spec.Replicas),
			ReadyReplicas:       s.Status.ReadyReplicas,
			AvailableReplicas:   s.Status.AvailableReplicas,
			Instrumented:        instr,
			InstrumentationType: instrType,
		})
	}
	return result, nil
}

func (wd *workloadDiscoverer) findDaemonSets(ns string) ([]WorkloadInfo, error) {
	ctx := context.Background()
	list, err := wd.clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing daemonsets in %q: %w", ns, err)
	}

	result := make([]WorkloadInfo, 0, len(list.Items))
	for _, ds := range list.Items {
		instr, instrType := detectTemplateInstrumentation(ds.Spec.Template)
		result = append(result, WorkloadInfo{
			Kind:              KindDaemonSet,
			Name:              ds.Name,
			Namespace:         ds.Namespace,
			Cluster:           wd.clusterName,
			Labels:            ds.Labels,
			Annotations:       ds.Annotations,
			Images:            containerImages(ds.Spec.Template.Spec),
			Replicas:          ds.Status.DesiredNumberScheduled,
			ReadyReplicas:     ds.Status.NumberReady,
			AvailableReplicas: ds.Status.NumberAvailable,
			Instrumented:      instr,
			InstrumentationType: instrType,
		})
	}
	return result, nil
}

// detectTemplateInstrumentation checks a pod template's annotations and container
// env vars — same signals as detectPodInstrumentation but works on templates,
// so workload-level discovery doesn't need to fetch live pods.
func detectTemplateInstrumentation(tmpl corev1.PodTemplateSpec) (bool, string) {
	for k := range tmpl.Annotations {
		if strings.HasPrefix(k, "instrumentation.opentelemetry.io/") {
			return true, InstrumentationOTel
		}
		if k == "newrelic.com/integrations-config" {
			return true, InstrumentationNROnHost
		}
	}

	allContainers := append(tmpl.Spec.Containers, tmpl.Spec.InitContainers...)
	for _, c := range allContainers {
		for _, env := range c.Env {
			switch env.Name {
			case "NEW_RELIC_LICENSE_KEY", "NEW_RELIC_APP_NAME":
				return true, InstrumentationNRAPM
			case "OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_SERVICE_NAME":
				return true, InstrumentationOTel
			}
		}
	}

	return false, InstrumentationNone
}

// containerImages returns the unique set of images from all containers + initContainers.
func containerImages(spec corev1.PodSpec) []string {
	seen := make(map[string]struct{})
	var images []string
	for _, c := range append(spec.Containers, spec.InitContainers...) {
		if _, ok := seen[c.Image]; !ok {
			seen[c.Image] = struct{}{}
			images = append(images, c.Image)
		}
	}
	return images
}

func derefInt32(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

