package discovery

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
				kubelet:    fakeKubelet(),
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
		kubelet:    fakeKubelet(),
	}
	result, err := d.Run()
	assert.NoError(t, err)

	assert.Len(t, result, 1)
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

func fakeKubelet() kubernetes.Kubelet {
	pod1 := corev1.Pod{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test",
			Namespace:   "test",
			ClusterName: "test",
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
			Name:        "fake",
			Namespace:   "fake",
			ClusterName: "fake",
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

	podList := corev1.PodList{
		TypeMeta: metav1.TypeMeta{},
		ListMeta: metav1.ListMeta{},
		Items:    []corev1.Pod{pod1, pod2},
	}

	client := fakeHttpClient(podList)
	k, _ := kubernetes.NewKubeletWithClient(&client)
	return k
}

func fakeHttpClient(pods corev1.PodList) http.HttpClient {
	return &FakeHttpClient{pods: pods}
}

type FakeHttpClient struct {
	pods corev1.PodList
}

func (k *FakeHttpClient) Get(path string) ([]byte, error) {
	return json.Marshal(k.pods)
}

func items() map[string]DiscoveredItem {
	items := map[string]DiscoveredItem{
		"test": {
			Variables: VariablesMap{
				cluster:                   "",
				node:                      "",
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
				node:                 "",
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
				node:                      "",
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
				node:                 "",
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
