package kubernetes

import (
	"context"
	"fmt"

	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// CRDInfo holds discovery-specific data for a CustomResourceDefinition.
// CRDs are cluster-scoped (not namespaced), so namespace filtering is ignored.
type CRDInfo struct {
	// Name is the full CRD name, e.g. "foos.example.com"
	Name        string
	Group       string
	Scope       string // Cluster | Namespaced
	Version     string // first served version
	Labels      LabelsMap
	Annotations AnnotationsMap
	Established bool // true when the CRD condition "Established" is True
}

// CRDDiscoverer discovers CustomResourceDefinitions.
type CRDDiscoverer interface {
	FindCRDs() ([]CRDInfo, error)
}

type crdDiscoverer struct {
	client *apiextclient.Clientset
}

// NewCRDDiscoverer creates a CRD discoverer using the provided REST config.
// Note: the standard k8s clientset does NOT include CRDs; a separate
// apiextensions clientset is required.
func NewCRDDiscoverer(restConfig *rest.Config) (CRDDiscoverer, error) {
	client, err := apiextclient.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("building apiextensions client: %w", err)
	}
	return &crdDiscoverer{client: client}, nil
}

func (cd *crdDiscoverer) FindCRDs() ([]CRDInfo, error) {
	ctx := context.Background()
	list, err := cd.client.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing CRDs: %w", err)
	}
	return transformCRDs(list.Items), nil
}

func transformCRDs(crds []apiextv1.CustomResourceDefinition) []CRDInfo {
	result := make([]CRDInfo, 0, len(crds))
	for _, c := range crds {
		version := ""
		for _, v := range c.Spec.Versions {
			if v.Served {
				version = v.Name
				break
			}
		}

		established := false
		for _, cond := range c.Status.Conditions {
			if cond.Type == apiextv1.Established && cond.Status == apiextv1.ConditionTrue {
				established = true
				break
			}
		}

		result = append(result, CRDInfo{
			Name:        c.Name,
			Group:       c.Spec.Group,
			Scope:       string(c.Spec.Scope),
			Version:     version,
			Labels:      c.Labels,
			Annotations: c.Annotations,
			Established: established,
		})
	}
	return result
}
