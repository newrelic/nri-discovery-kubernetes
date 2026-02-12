package discovery

import (
	"testing"

	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testServiceClusterName = "test-service-cluster"
)

// Note: Testing with fake Kubernetes clientset is challenging due to type incompatibility
// between *fake.Clientset and *kubernetes.Clientset in the production code.
// These tests focus on the processServices function which contains the core discovery logic.

func TestProcessServices(t *testing.T) {
	tests := []struct {
		name         string
		services     []kubernetes.ServiceInfo
		wantCount    int
		validateFunc func(t *testing.T, output Output)
	}{
		{
			name: "process single ClusterIP service",
			services: []kubernetes.ServiceInfo{
				createServiceInfo("nginx", "default", "ClusterIP", "10.96.0.1"),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.Equal(t, testServiceClusterName, item.Variables[cluster])
				assert.Equal(t, "default", item.Variables[namespace])
				assert.Equal(t, "nginx", item.Variables[serviceName])
				assert.Equal(t, "ClusterIP", item.Variables[serviceType])
				assert.Equal(t, "10.96.0.1", item.Variables[clusterIP])
			},
		},
		{
			name: "process NodePort service",
			services: []kubernetes.ServiceInfo{
				createNodePortServiceInfo(),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.Equal(t, "NodePort", item.Variables[serviceType])
				assert.NotNil(t, item.Variables[ports])
				ports := item.Variables[ports].([]kubernetes.ServicePortInfo)
				assert.Len(t, ports, 1)
				assert.Equal(t, int32(30080), ports[0].NodePort)
			},
		},
		{
			name: "process LoadBalancer service with external IPs",
			services: []kubernetes.ServiceInfo{
				createLoadBalancerServiceInfo(),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.Equal(t, "LoadBalancer", item.Variables[serviceType])
				assert.Contains(t, item.Variables, externalIPs)
				ips := item.Variables[externalIPs].([]string)
				assert.Len(t, ips, 2)
				assert.Contains(t, ips, "192.168.1.1")
				assert.Contains(t, ips, "192.168.1.2")
			},
		},
		{
			name: "process service with labels",
			services: []kubernetes.ServiceInfo{
				createServiceWithLabels(),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.Equal(t, "nginx", item.Variables[labelPrefix+"app"])
				assert.Equal(t, "production", item.Variables[labelPrefix+"environment"])
			},
		},
		{
			name: "process service with annotations",
			services: []kubernetes.ServiceInfo{
				createServiceWithAnnotations(),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.Equal(t, "true", item.Variables[annotationPrefix+"prometheus.io/scrape"])
				assert.Equal(t, "/metrics", item.Variables[annotationPrefix+"prometheus.io/path"])
			},
		},
		{
			name: "process service with selector",
			services: []kubernetes.ServiceInfo{
				createServiceWithSelector(),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.Equal(t, "nginx", item.Variables[labelPrefix+"app"])
				assert.Equal(t, "backend", item.Variables[labelPrefix+"tier"])
			},
		},
		{
			name: "process service with ports",
			services: []kubernetes.ServiceInfo{
				createServiceWithPorts(),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.Contains(t, item.Variables, ports)
				servicePorts := item.Variables[ports].([]kubernetes.ServicePortInfo)
				assert.Len(t, servicePorts, 2)
				assert.Equal(t, "http", servicePorts[0].Name)
				assert.Equal(t, int32(80), servicePorts[0].Port)
				assert.Equal(t, "8080", servicePorts[0].TargetPort)
				assert.Equal(t, "TCP", servicePorts[0].Protocol)
			},
		},
		{
			name: "process service without external IPs",
			services: []kubernetes.ServiceInfo{
				createServiceInfo("test", "default", "ClusterIP", "10.96.0.1"),
			},
			wantCount: 1,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				item := output[0]
				assert.NotContains(t, item.Variables, externalIPs)
			},
		},
		{
			name:      "process empty service list",
			services:  []kubernetes.ServiceInfo{},
			wantCount: 0,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				assert.Empty(t, output)
			},
		},
		{
			name: "process multiple services",
			services: []kubernetes.ServiceInfo{
				createServiceInfo("service1", "default", "ClusterIP", "10.96.0.1"),
				createServiceInfo("service2", "kube-system", "NodePort", "10.96.0.2"),
				createServiceInfo("service3", "default", "LoadBalancer", "10.96.0.3"),
			},
			wantCount: 3,
			validateFunc: func(t *testing.T, output Output) {
				t.Helper()
				names := make(map[string]bool)
				for _, item := range output {
					names[item.Variables[serviceName].(string)] = true
				}
				assert.True(t, names["service1"])
				assert.True(t, names["service2"])
				assert.True(t, names["service3"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := processServices(tt.services)

			assert.Len(t, output, tt.wantCount)
			if tt.validateFunc != nil {
				tt.validateFunc(t, output)
			}
		})
	}
}

func TestProcessServices_EntityRewrites(t *testing.T) {
	services := []kubernetes.ServiceInfo{
		createServiceInfo("test-service", "default", "ClusterIP", "10.96.0.1"),
	}

	output := processServices(services)

	require.Len(t, output, 1)
	item := output[0]

	// Verify entity rewrites
	require.Len(t, item.EntityRewrites, 1)
	rewrite := item.EntityRewrites[0]

	assert.Equal(t, entityRewriteActionReplace, rewrite.Action)
	assert.Equal(t, "${clusterIP}", rewrite.Match)
	assert.Equal(t, serviceEntityReplaceField, rewrite.ReplaceField)
}

func TestProcessServices_MetricAnnotations(t *testing.T) {
	tests := []struct {
		name         string
		service      kubernetes.ServiceInfo
		validateFunc func(t *testing.T, annotations AnnotationsMap)
	}{
		{
			name:    "metric annotations exclude annotation prefix",
			service: createServiceWithAnnotations(),
			validateFunc: func(t *testing.T, annotations AnnotationsMap) {
				t.Helper()
				// Annotations with prefix should not be in metric annotations
				for key := range annotations {
					assert.NotContains(t, key, annotationPrefix)
				}
			},
		},
		{
			name:    "metric annotations include cluster and namespace",
			service: createServiceInfo("test", "default", "ClusterIP", "10.96.0.1"),
			validateFunc: func(t *testing.T, annotations AnnotationsMap) {
				t.Helper()
				assert.Contains(t, annotations, cluster)
				assert.Contains(t, annotations, namespace)
				assert.Equal(t, testServiceClusterName, annotations[cluster])
				assert.Equal(t, "default", annotations[namespace])
			},
		},
		{
			name:    "metric annotations include service name and type",
			service: createServiceInfo("nginx", "default", "NodePort", "10.96.0.1"),
			validateFunc: func(t *testing.T, annotations AnnotationsMap) {
				t.Helper()
				assert.Contains(t, annotations, serviceName)
				assert.Contains(t, annotations, serviceType)
				assert.Equal(t, "nginx", annotations[serviceName])
				assert.Equal(t, "NodePort", annotations[serviceType])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := processServices([]kubernetes.ServiceInfo{tt.service})

			require.Len(t, output, 1)
			item := output[0]
			if tt.validateFunc != nil {
				tt.validateFunc(t, item.MetricAnnotations)
			}
		})
	}
}

func TestProcessServices_AllFieldsPresent(t *testing.T) {
	service := kubernetes.ServiceInfo{
		Name:      "complete-service",
		Namespace: "production",
		Type:      "LoadBalancer",
		ClusterIP: "10.96.0.10",
		ExternalIPs: []string{
			"203.0.113.1",
			"203.0.113.2",
		},
		Ports: []kubernetes.ServicePortInfo{
			{
				Name:       "http",
				Port:       80,
				TargetPort: "8080",
				Protocol:   "TCP",
			},
			{
				Name:       "https",
				Port:       443,
				TargetPort: "8443",
				Protocol:   "TCP",
				NodePort:   30443,
			},
		},
		Selector: kubernetes.LabelsMap{
			"app":     "web",
			"version": "v1",
		},
		Labels: kubernetes.LabelsMap{
			"environment": "production",
			"team":        "platform",
		},
		Annotations: kubernetes.AnnotationsMap{
			"prometheus.io/scrape": "true",
			"prometheus.io/port":   "9090",
		},
		Cluster: testServiceClusterName,
	}

	output := processServices([]kubernetes.ServiceInfo{service})

	require.Len(t, output, 1)
	item := output[0]

	// Verify all core fields
	assert.Equal(t, testServiceClusterName, item.Variables[cluster])
	assert.Equal(t, "production", item.Variables[namespace])
	assert.Equal(t, "complete-service", item.Variables[serviceName])
	assert.Equal(t, "LoadBalancer", item.Variables[serviceType])
	assert.Equal(t, "10.96.0.10", item.Variables[clusterIP])

	// Verify external IPs
	assert.Contains(t, item.Variables, externalIPs)
	ips := item.Variables[externalIPs].([]string)
	assert.Len(t, ips, 2)

	// Verify ports
	assert.Contains(t, item.Variables, ports)
	servicePorts := item.Variables[ports].([]kubernetes.ServicePortInfo)
	assert.Len(t, servicePorts, 2)

	// Verify selector labels
	assert.Equal(t, "web", item.Variables[labelPrefix+"app"])
	assert.Equal(t, "v1", item.Variables[labelPrefix+"version"])

	// Verify service labels
	assert.Equal(t, "production", item.Variables[labelPrefix+"environment"])
	assert.Equal(t, "platform", item.Variables[labelPrefix+"team"])

	// Verify annotations
	assert.Equal(t, "true", item.Variables[annotationPrefix+"prometheus.io/scrape"])
	assert.Equal(t, "9090", item.Variables[annotationPrefix+"prometheus.io/port"])

	// Verify entity rewrites
	require.Len(t, item.EntityRewrites, 1)
	assert.Equal(t, entityRewriteActionReplace, item.EntityRewrites[0].Action)

	// Verify metric annotations are properly filtered
	assert.NotEmpty(t, item.MetricAnnotations)
}

func TestProcessServices_NamespaceFiltering(t *testing.T) {
	// Test that services from different namespaces are correctly processed
	tests := []struct {
		name           string
		services       []kubernetes.ServiceInfo
		wantNamespaces []string
	}{
		{
			name: "services from multiple namespaces",
			services: []kubernetes.ServiceInfo{
				createServiceInfo("svc1", "default", "ClusterIP", "10.96.0.1"),
				createServiceInfo("svc2", "kube-system", "ClusterIP", "10.96.0.2"),
				createServiceInfo("svc3", "production", "ClusterIP", "10.96.0.3"),
			},
			wantNamespaces: []string{"default", "kube-system", "production"},
		},
		{
			name: "services from same namespace",
			services: []kubernetes.ServiceInfo{
				createServiceInfo("svc1", "default", "ClusterIP", "10.96.0.1"),
				createServiceInfo("svc2", "default", "ClusterIP", "10.96.0.2"),
			},
			wantNamespaces: []string{"default", "default"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := processServices(tt.services)

			require.Len(t, output, len(tt.services))

			// Verify namespaces are preserved correctly
			for i, item := range output {
				assert.Equal(t, tt.wantNamespaces[i], item.Variables[namespace])
			}
		})
	}
}

// Helper functions to create test service data

func createServiceInfo(name, namespace, svcType, clusterIP string) kubernetes.ServiceInfo {
	return kubernetes.ServiceInfo{
		Name:      name,
		Namespace: namespace,
		Type:      svcType,
		ClusterIP: clusterIP,
		Cluster:   testServiceClusterName,
	}
}

func createNodePortServiceInfo() kubernetes.ServiceInfo {
	return kubernetes.ServiceInfo{
		Name:      "nodeport-service",
		Namespace: "default",
		Type:      "NodePort",
		ClusterIP: "10.96.0.2",
		Ports: []kubernetes.ServicePortInfo{
			{
				Name:       "http",
				Port:       80,
				TargetPort: "8080",
				Protocol:   "TCP",
				NodePort:   30080,
			},
		},
		Cluster: testServiceClusterName,
	}
}

func createLoadBalancerServiceInfo() kubernetes.ServiceInfo {
	return kubernetes.ServiceInfo{
		Name:      "lb-service",
		Namespace: "kube-system",
		Type:      "LoadBalancer",
		ClusterIP: "10.96.0.3",
		ExternalIPs: []string{
			"192.168.1.1",
			"192.168.1.2",
		},
		Ports: []kubernetes.ServicePortInfo{
			{
				Name:       "http",
				Port:       80,
				TargetPort: "http",
				Protocol:   "TCP",
			},
		},
		Cluster: testServiceClusterName,
	}
}

func createServiceWithLabels() kubernetes.ServiceInfo {
	return kubernetes.ServiceInfo{
		Name:      "labeled-service",
		Namespace: "production",
		Type:      "ClusterIP",
		ClusterIP: "10.96.0.4",
		Labels: kubernetes.LabelsMap{
			"app":         "nginx",
			"environment": "production",
		},
		Cluster: testServiceClusterName,
	}
}

func createServiceWithAnnotations() kubernetes.ServiceInfo {
	return kubernetes.ServiceInfo{
		Name:      "annotated-service",
		Namespace: "default",
		Type:      "ClusterIP",
		ClusterIP: "10.96.0.5",
		Annotations: kubernetes.AnnotationsMap{
			"prometheus.io/scrape": "true",
			"prometheus.io/path":   "/metrics",
		},
		Cluster: testServiceClusterName,
	}
}

func createServiceWithSelector() kubernetes.ServiceInfo {
	return kubernetes.ServiceInfo{
		Name:      "selector-service",
		Namespace: "staging",
		Type:      "ClusterIP",
		ClusterIP: "10.96.0.6",
		Selector: kubernetes.LabelsMap{
			"app":  "nginx",
			"tier": "backend",
		},
		Cluster: testServiceClusterName,
	}
}

func createServiceWithPorts() kubernetes.ServiceInfo {
	return kubernetes.ServiceInfo{
		Name:      "ports-service",
		Namespace: "default",
		Type:      "ClusterIP",
		ClusterIP: "10.96.0.7",
		Ports: []kubernetes.ServicePortInfo{
			{
				Name:       "http",
				Port:       80,
				TargetPort: "8080",
				Protocol:   "TCP",
			},
			{
				Name:       "https",
				Port:       443,
				TargetPort: "8443",
				Protocol:   "TCP",
			},
		},
		Cluster: testServiceClusterName,
	}
}