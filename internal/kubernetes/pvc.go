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

// PVCInfo holds discovery-specific data for a PersistentVolumeClaim.
type PVCInfo struct {
	Name         string
	Namespace    string
	Cluster      string
	Labels       LabelsMap
	Annotations  AnnotationsMap
	StorageClass string
	AccessModes  []string
	Capacity     string // human-readable storage size, e.g. "10Gi"
	Status       string // Bound | Pending | Lost
	VolumeName   string // PV name when Bound
}

// PVCDiscoverer discovers PersistentVolumeClaims.
type PVCDiscoverer interface {
	FindPVCs(namespaces []string) ([]PVCInfo, error)
}

type pvcDiscoverer struct {
	clientset   *kubernetes.Clientset
	clusterName string
}

// NewPVCDiscoverer creates a PVC discoverer.
func NewPVCDiscoverer(clientset *kubernetes.Clientset, cfg *config.Config) PVCDiscoverer {
	return &pvcDiscoverer{clientset: clientset, clusterName: cfg.ClusterName}
}

func (pd *pvcDiscoverer) FindPVCs(namespaces []string) ([]PVCInfo, error) {
	ctx := context.Background()
	var allPVCs []corev1.PersistentVolumeClaim

	if len(namespaces) == 0 {
		list, err := pd.clientset.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing PVCs: %w", err)
		}
		allPVCs = list.Items
	} else {
		for _, ns := range namespaces {
			list, err := pd.clientset.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
			if err != nil {
				return nil, fmt.Errorf("listing PVCs in %q: %w", ns, err)
			}
			allPVCs = append(allPVCs, list.Items...)
		}
	}

	return transformPVCs(pd.clusterName, allPVCs), nil
}

func transformPVCs(clusterName string, pvcs []corev1.PersistentVolumeClaim) []PVCInfo {
	result := make([]PVCInfo, 0, len(pvcs))
	for _, p := range pvcs {
		modes := make([]string, len(p.Spec.AccessModes))
		for i, m := range p.Spec.AccessModes {
			modes[i] = string(m)
		}

		capacity := ""
		if storage, ok := p.Status.Capacity[corev1.ResourceStorage]; ok {
			capacity = storage.String()
		}

		storageClass := ""
		if p.Spec.StorageClassName != nil {
			storageClass = *p.Spec.StorageClassName
		}

		result = append(result, PVCInfo{
			Name:         p.Name,
			Namespace:    p.Namespace,
			Cluster:      clusterName,
			Labels:       p.Labels,
			Annotations:  p.Annotations,
			StorageClass: storageClass,
			AccessModes:  modes,
			Capacity:     capacity,
			Status:       string(p.Status.Phase),
			VolumeName:   p.Spec.VolumeName,
		})
	}
	return result
}

// accessModesString formats access modes as a comma-separated string.
func accessModesString(modes []string) string {
	return strings.Join(modes, ",")
}
