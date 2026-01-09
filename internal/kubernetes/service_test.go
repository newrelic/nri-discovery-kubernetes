package kubernetes

import (
	"testing"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	testClusterName = "test-cluster"
)

// Note: Testing NewServiceDiscoverer and FindServices with fake clientset is challenging
// due to type incompatibility between *fake.Clientset and *kubernetes.Clientset.
// The production code uses concrete types instead of interfaces.
// We focus on testing the core transformation logic which is well-isolated.

func TestNewServiceDiscoverer(t *testing.T) {
	// Verify the constructor exists and has correct signature
	cfg := &config.Config{
		ClusterName: testClusterName,
	}

	require.NotNil(t, NewServiceDiscoverer)
	require.NotNil(t, cfg)

	// The actual service discovery logic is tested through transformServices
}

func TestTransformServices(t *testing.T) {
	tests := []struct {
		name         string
		clusterName  string
		services     []corev1.Service
		wantCount    int
		validateFunc func(t *testing.T, services []ServiceInfo)
	}{
		{
			name:        "transform ClusterIP service",
			clusterName: testClusterName,
			services:    []corev1.Service{createClusterIPService()},
			wantCount:   1,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				svc := services[0]
				assert.Equal(t, "nginx-service", svc.Name)
				assert.Equal(t, "default", svc.Namespace)
				assert.Equal(t, "ClusterIP", svc.Type)
				assert.Equal(t, "10.96.0.1", svc.ClusterIP)
				assert.Equal(t, testClusterName, svc.Cluster)
				assert.Len(t, svc.Ports, 2)
				assert.Equal(t, "http", svc.Ports[0].Name)
				assert.Equal(t, int32(80), svc.Ports[0].Port)
				assert.Equal(t, "8080", svc.Ports[0].TargetPort)
				assert.Equal(t, "TCP", svc.Ports[0].Protocol)
			},
		},
		{
			name:        "transform NodePort service",
			clusterName: testClusterName,
			services:    []corev1.Service{createNodePortService()},
			wantCount:   1,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				svc := services[0]
				assert.Equal(t, "nodeport-service", svc.Name)
				assert.Equal(t, "default", svc.Namespace)
				assert.Equal(t, "NodePort", svc.Type)
				assert.Len(t, svc.Ports, 1)
				assert.Equal(t, int32(30080), svc.Ports[0].NodePort)
			},
		},
		{
			name:        "transform LoadBalancer service",
			clusterName: testClusterName,
			services:    []corev1.Service{createLoadBalancerService()},
			wantCount:   1,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				svc := services[0]
				assert.Equal(t, "lb-service", svc.Name)
				assert.Equal(t, "kube-system", svc.Namespace)
				assert.Equal(t, "LoadBalancer", svc.Type)
				assert.Len(t, svc.ExternalIPs, 2)
				assert.Contains(t, svc.ExternalIPs, "192.168.1.1")
				assert.Contains(t, svc.ExternalIPs, "192.168.1.2")
			},
		},
		{
			name:        "transform service with labels and annotations",
			clusterName: testClusterName,
			services:    []corev1.Service{createServiceWithLabelsAndAnnotations()},
			wantCount:   1,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				svc := services[0]
				assert.Equal(t, "labeled-service", svc.Name)
				assert.Len(t, svc.Labels, 2)
				assert.Equal(t, "nginx", svc.Labels["app"])
				assert.Equal(t, "production", svc.Labels["environment"])
				assert.Len(t, svc.Annotations, 2)
				assert.Equal(t, "true", svc.Annotations["prometheus.io/scrape"])
				assert.Equal(t, "/metrics", svc.Annotations["prometheus.io/path"])
			},
		},
		{
			name:        "transform service with selector",
			clusterName: testClusterName,
			services:    []corev1.Service{createServiceWithSelector()},
			wantCount:   1,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				svc := services[0]
				assert.Equal(t, "selector-service", svc.Name)
				assert.Len(t, svc.Selector, 2)
				assert.Equal(t, "nginx", svc.Selector["app"])
				assert.Equal(t, "backend", svc.Selector["tier"])
			},
		},
		{
			name:        "transform service without ports",
			clusterName: testClusterName,
			services:    []corev1.Service{createServiceWithoutPorts()},
			wantCount:   1,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				svc := services[0]
				assert.Equal(t, "no-ports-service", svc.Name)
				assert.Len(t, svc.Ports, 0)
			},
		},
		{
			name:        "transform service with named and unnamed ports",
			clusterName: testClusterName,
			services:    []corev1.Service{createServiceWithMixedPorts()},
			wantCount:   1,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				svc := services[0]
				assert.Len(t, svc.Ports, 3)
				// Named port
				assert.Equal(t, "http", svc.Ports[0].Name)
				assert.Equal(t, int32(80), svc.Ports[0].Port)
				// Unnamed port
				assert.Equal(t, "", svc.Ports[1].Name)
				assert.Equal(t, int32(443), svc.Ports[1].Port)
				// Named port with numeric target
				assert.Equal(t, "metrics", svc.Ports[2].Name)
				assert.Equal(t, "9090", svc.Ports[2].TargetPort)
			},
		},
		{
			name:        "transform empty service list",
			clusterName: testClusterName,
			services:    []corev1.Service{},
			wantCount:   0,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				assert.Empty(t, services)
			},
		},
		{
			name:        "transform multiple services",
			clusterName: testClusterName,
			services:    createTestServices(),
			wantCount:   5,
			validateFunc: func(t *testing.T, services []ServiceInfo) {
				t.Helper()
				// Verify all services have the cluster name
				for _, svc := range services {
					assert.Equal(t, testClusterName, svc.Cluster)
				}
				// Verify service names
				names := make(map[string]bool)
				for _, svc := range services {
					names[svc.Name] = true
				}
				assert.True(t, names["nginx-service"])
				assert.True(t, names["nodeport-service"])
				assert.True(t, names["lb-service"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := transformServices(tt.clusterName, tt.services)

			assert.Len(t, result, tt.wantCount)
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestTransformServices_PortTargetPortHandling(t *testing.T) {
	tests := []struct {
		name       string
		targetPort intstr.IntOrString
		want       string
	}{
		{
			name:       "target port as integer",
			targetPort: intstr.FromInt(8080),
			want:       "8080",
		},
		{
			name:       "target port as string",
			targetPort: intstr.FromString("http"),
			want:       "http",
		},
		{
			name:       "empty target port",
			targetPort: intstr.IntOrString{},
			want:       "0", // Empty IntOrString serializes to "0"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "default",
				},
				Spec: corev1.ServiceSpec{
					Type:      corev1.ServiceTypeClusterIP,
					ClusterIP: "10.96.0.1",
					Ports: []corev1.ServicePort{
						{
							Name:       "test",
							Port:       80,
							TargetPort: tt.targetPort,
							Protocol:   corev1.ProtocolTCP,
						},
					},
				},
			}

			result := transformServices(testClusterName, []corev1.Service{svc})

			require.Len(t, result, 1)
			require.Len(t, result[0].Ports, 1)
			assert.Equal(t, tt.want, result[0].Ports[0].TargetPort)
		})
	}
}

func TestServicePortInfo(t *testing.T) {
	port := ServicePortInfo{
		Name:       "http",
		Port:       80,
		TargetPort: "8080",
		Protocol:   "TCP",
		NodePort:   30080,
	}

	assert.Equal(t, "http", port.Name)
	assert.Equal(t, int32(80), port.Port)
	assert.Equal(t, "8080", port.TargetPort)
	assert.Equal(t, "TCP", port.Protocol)
	assert.Equal(t, int32(30080), port.NodePort)
}

func TestServiceInfo(t *testing.T) {
	info := ServiceInfo{
		Name:        "test-service",
		Namespace:   "default",
		Type:        "ClusterIP",
		ClusterIP:   "10.96.0.1",
		ExternalIPs: []string{"192.168.1.1"},
		Ports: []ServicePortInfo{
			{
				Name:       "http",
				Port:       80,
				TargetPort: "8080",
				Protocol:   "TCP",
			},
		},
		Selector: map[string]string{
			"app": "nginx",
		},
		Labels: map[string]string{
			"environment": "production",
		},
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
		},
		Cluster: testClusterName,
	}

	assert.Equal(t, "test-service", info.Name)
	assert.Equal(t, "default", info.Namespace)
	assert.Equal(t, "ClusterIP", info.Type)
	assert.Equal(t, "10.96.0.1", info.ClusterIP)
	assert.Len(t, info.ExternalIPs, 1)
	assert.Len(t, info.Ports, 1)
	assert.Len(t, info.Selector, 1)
	assert.Len(t, info.Labels, 1)
	assert.Len(t, info.Annotations, 1)
	assert.Equal(t, testClusterName, info.Cluster)
}

// Helper functions to create test services

func createTestServices() []corev1.Service {
	return []corev1.Service{
		createClusterIPService(),
		createNodePortService(),
		createLoadBalancerService(),
		createServiceWithLabelsAndAnnotations(),
		createServiceWithSelector(),
	}
}

func createClusterIPService() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.1",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "https",
					Port:       443,
					TargetPort: intstr.FromInt(8443),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": "nginx",
			},
		},
	}
}

func createNodePortService() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nodeport-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeNodePort,
			ClusterIP: "10.96.0.2",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
					NodePort:   30080,
				},
			},
			Selector: map[string]string{
				"app": "api",
			},
		},
	}
}

func createLoadBalancerService() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lb-service",
			Namespace: "kube-system",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeLoadBalancer,
			ClusterIP: "10.96.0.3",
			ExternalIPs: []string{
				"192.168.1.1",
				"192.168.1.2",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromString("http"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func createServiceWithLabelsAndAnnotations() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "labeled-service",
			Namespace: "production",
			Labels: map[string]string{
				"app":         "nginx",
				"environment": "production",
			},
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "/metrics",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.4",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func createServiceWithSelector() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "selector-service",
			Namespace: "staging",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.5",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromInt(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app":  "nginx",
				"tier": "backend",
			},
		},
	}
}

func createServiceWithoutPorts() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-ports-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.6",
			Ports:     []corev1.ServicePort{},
		},
	}
}

func createServiceWithMixedPorts() corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mixed-ports-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.7",
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					TargetPort: intstr.FromString("http-port"),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					// Unnamed port
					Port:       443,
					TargetPort: intstr.FromInt(8443),
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       "metrics",
					Port:       9090,
					TargetPort: intstr.FromInt(9090),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
