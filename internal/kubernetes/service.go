package kubernetes

import (
	"context"
	"fmt"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ServicePortInfo represents a service port.
type ServicePortInfo struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort"`
	Protocol   string `json:"protocol"`
	NodePort   int32  `json:"nodePort,omitempty"`
}

// ServiceInfo represents discovery-specific format for found Services via Kubernetes API.
type ServiceInfo struct {
	Name            string
	Namespace       string
	Type            string
	ClusterIP       string
	ExternalIPs    []string
	Ports           []ServicePortInfo
	Selector        LabelsMap
	Labels          LabelsMap
	Annotations     AnnotationsMap
	Cluster         string
}

// ServiceDiscoverer defines what functionality service discovery client provides.
type ServiceDiscoverer interface {
	FindServices(namespaces []string) ([]ServiceInfo, error)
}

type serviceDiscoverer struct {
	clientset   *kubernetes.Clientset
	ClusterName string
}

func (sd *serviceDiscoverer) FindServices(namespaces []string) ([]ServiceInfo, error) {
	allServices, err := sd.getServices(namespaces)
	if err != nil {
		return nil, err
	}
	return transformServices(sd.ClusterName, allServices), nil
}

func (sd *serviceDiscoverer) getServices(namespaces []string) ([]corev1.Service, error) {
	ctx := context.Background()
	var allServices []corev1.Service

	// If no namespaces specified, get from all namespaces
	if len(namespaces) == 0 {
		serviceList, err := sd.clientset.CoreV1().Services(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list services: %w", err)
		}
		return serviceList.Items, nil
	}

	// Get services from specified namespaces
	for _, ns := range namespaces {
		serviceList, err := sd.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list services in namespace %s: %w", ns, err)
		}
		allServices = append(allServices, serviceList.Items...)
	}

	return allServices, nil
}

func transformServices(clusterName string, services []corev1.Service) []ServiceInfo {
	var result []ServiceInfo

	for _, svc := range services {
		ports := make([]ServicePortInfo, len(svc.Spec.Ports))
		for i, port := range svc.Spec.Ports {
			targetPort := ""
			if port.TargetPort.String() != "" {
				targetPort = port.TargetPort.String()
			}
			ports[i] = ServicePortInfo{
				Name:       port.Name,
				Port:       port.Port,
				TargetPort: targetPort,
				Protocol:   string(port.Protocol),
				NodePort:   port.NodePort,
			}
		}

		serviceInfo := ServiceInfo{
			Name:          svc.Name,
			Namespace:     svc.Namespace,
			Type:          string(svc.Spec.Type),
			ClusterIP:     svc.Spec.ClusterIP,
			ExternalIPs:   svc.Spec.ExternalIPs,
			Ports:         ports,
			Selector:      svc.Spec.Selector,
			Labels:        svc.Labels,
			Annotations:   svc.Annotations,
			Cluster:       clusterName,
		}
		result = append(result, serviceInfo)
	}

	return result
}

// NewServiceDiscoverer creates a new service discoverer using the provided clientset.
func NewServiceDiscoverer(clientset *kubernetes.Clientset, config *config.Config) ServiceDiscoverer {
	return &serviceDiscoverer{
		clientset:   clientset,
		ClusterName: config.ClusterName,
	}
}
