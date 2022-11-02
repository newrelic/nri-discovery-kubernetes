package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	internalhttp "github.com/newrelic/nri-discovery-kubernetes/internal/http"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
)

const (
	nodeName      = "test-node"
	fakeTokenFile = "./test_data/token" //nolint: gosec  // testing credentials
)

func TestDiscoverer_Run(t *testing.T) {
	type fields struct {
		namespaces []string
	}
	tests := []struct {
		name    string
		fields  fields
		want    Output
		wantErr bool
	}{
		{
			name:    "Test_Empty_Namespaces_Returns_All_Pods",
			fields:  struct{ namespaces []string }{namespaces: nil},
			want:    allItems(),
			wantErr: false,
		},
		{
			name:    "Test_Single_Namespace_Returns_Single_Pod",
			fields:  struct{ namespaces []string }{namespaces: []string{"test"}},
			want:    singleItem("test"),
			wantErr: false,
		},
		{
			name:    "Test_NonExisting_Namespace_Returns_Empty",
			fields:  struct{ namespaces []string }{namespaces: []string{"invalid"}},
			want:    noItem(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Discoverer{
				namespaces: tt.fields.namespaces,
				kubelet:    fakeKubeletClient(t),
			}
			got, err := d.Run()
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualValues(t, got, tt.want)
		})
	}
}

func Test_PodsWithMultiplePorts_ReturnsIndexAndName(t *testing.T) {
	d := &Discoverer{
		namespaces: []string{"test"},
		kubelet:    fakeKubeletClient(t),
	}
	result, err := d.Run()
	require.NoError(t, err)

	require.Len(t, result, 1)
	assert.NotEmpty(t, result[0].Variables)
	assert.Contains(t, result[0].Variables, ports)

	// assert correct type
	p := result[0].Variables[ports].(kubernetes.PortsMap)
	assert.NotEmpty(t, p)

	assert.Contains(t, p, "0")
	assert.Contains(t, p, "first")
	assert.EqualValues(t, p["0"], p["first"])

	assert.Contains(t, p, "1")

	assert.Contains(t, p, "2")
	assert.Contains(t, p, "third")
	assert.EqualValues(t, p["2"], p["third"])
}

func fakeKubeletClient(t *testing.T) kubernetes.Kubelet {
	t.Helper()

	server := httptest.NewServer(fakePodListHandler(t))

	k8sClient, cf, inClusterConfig, logger := getTestData(server)
	httpClient, _ := internalhttp.NewClient(
		internalhttp.DefaultConnector(k8sClient, cf, inClusterConfig, logger),
		internalhttp.WithMaxRetries(5),
		internalhttp.WithLogger(logger),
	)

	return kubernetes.New(httpClient, cf)
}

func getTestData(s *httptest.Server) (*fake.Clientset, *config.Config, *rest.Config, *log.Logger) {
	u, _ := url.Parse(s.URL)
	port, _ := strconv.Atoi(u.Port())

	c := fake.NewSimpleClientset(getTestNode(port))

	cf := &config.Config{
		NodeName: nodeName,
		Host:     u.Hostname(),
	}

	inClusterConfig := &rest.Config{
		Host:            fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		BearerTokenFile: fakeTokenFile,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}

	logger := log.New()
	logger.SetOutput(io.Discard)
	// Set level to panic might save a few cycles if we don't even attempt to write to io.Discard.
	logger.SetLevel(log.PanicLevel)

	return c, cf, inClusterConfig, logger
}

func getTestNode(port int) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Status: v1.NodeStatus{
			DaemonEndpoints: v1.NodeDaemonEndpoints{
				KubeletEndpoint: v1.DaemonEndpoint{
					Port: int32(port),
				},
			},
		},
	}
}

func fakePodList() corev1.PodList {
	pod1 := corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels: map[string]string{
				"team": "caos",
			},
			Annotations: map[string]string{
				"test": "test",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Ports: []corev1.ContainerPort{
						{
							Name:          "first",
							ContainerPort: 1,
						},
						{
							ContainerPort: 2,
						},
						{
							Name:          "third",
							ContainerPort: 3,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase:  corev1.PodRunning,
			PodIP:  "127.0.0.1",
			HostIP: "10.0.0.0",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "test",
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					ContainerID: "testID",
					Image:       "testImage",
					ImageID:     "testImageID",
				},
			},
		},
	}
	pod2 := corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake",
			Namespace: "fake",
			Labels: map[string]string{
				"team": "caos",
			},
			Annotations: map[string]string{
				"fake": "fake",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 1,
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase:  corev1.PodRunning,
			PodIP:  "127.0.0.2",
			HostIP: "10.0.0.0",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "fake",
					State: corev1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
					ContainerID: "fakeID",
					Image:       "fakeImage",
					ImageID:     "fakeImageID",
				},
			},
		},
	}

	return corev1.PodList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items:    []corev1.Pod{pod1, pod2},
	}
}

func fakePodListHandler(t *testing.T) http.HandlerFunc {
	t.Helper()

	marshaledPodList, err := json.Marshal(fakePodList())
	require.NoError(t, err)

	return func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(marshaledPodList)
		require.NoError(t, err)
	}
}

func items() map[string]DiscoveredItem {
	items := map[string]DiscoveredItem{
		"test": {
			Variables: VariablesMap{
				cluster:                   "",
				node:                      nodeName,
				nodeIP:                    "10.0.0.0",
				namespace:                 "test",
				podName:                   "test",
				ip:                        "127.0.0.1",
				ports:                     kubernetes.PortsMap{"0": 1, "1": 2, "2": 3, "first": 1, "third": 3},
				name:                      "test",
				id:                        "testID",
				image:                     "testImage",
				labelPrefix + "team":      "caos",
				annotationPrefix + "test": "test",
			},
			MetricAnnotations: AnnotationsMap{
				cluster:              "",
				node:                 nodeName,
				namespace:            "test",
				podName:              "test",
				name:                 "test",
				image:                "testImage",
				labelPrefix + "team": "caos",
			},
			EntityRewrites: []Replacement{
				{
					Action:       "replace",
					Match:        "${ip}",
					ReplaceField: "k8s:${clusterName}:${namespace}:pod:${podName}:${name}",
				},
			},
		},
		"fake": {
			Variables: VariablesMap{
				cluster:                   "",
				node:                      nodeName,
				nodeIP:                    "10.0.0.0",
				namespace:                 "fake",
				podName:                   "fake",
				ip:                        "127.0.0.2",
				ports:                     kubernetes.PortsMap{"0": 1},
				name:                      "fake",
				id:                        "fakeID",
				image:                     "fakeImage",
				labelPrefix + "team":      "caos",
				annotationPrefix + "fake": "fake",
			},
			MetricAnnotations: AnnotationsMap{
				cluster:              "",
				node:                 nodeName,
				namespace:            "fake",
				podName:              "fake",
				name:                 "fake",
				image:                "fakeImage",
				labelPrefix + "team": "caos",
			},
			EntityRewrites: []Replacement{
				{
					Action:       "replace",
					Match:        "${ip}",
					ReplaceField: "k8s:${clusterName}:${namespace}:pod:${podName}:${name}",
				},
			},
		},
	}

	return items
}

func allItems() Output {
	discoveredItems := items()
	test := discoveredItems["test"]
	fake := discoveredItems["fake"]

	output := Output{
		test,
		fake,
	}
	return output
}

func singleItem(ns string) Output {
	discoveredItems := items()
	item := discoveredItems[ns]

	output := Output{
		item,
	}
	return output
}

func noItem() Output {
	return Output{}
}
