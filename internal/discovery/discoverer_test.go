package discovery

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func noItem() Output {
	return Output{}
}

func fakeKubelet() kubernetes.Kubelet {
	pod1 := v1.Pod{
		TypeMeta: v12.TypeMeta{},
		ObjectMeta: v12.ObjectMeta{
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
		Spec: v1.PodSpec{},
		Status: v1.PodStatus{
			PodIP:  "127.0.0.1",
			HostIP: "10.0.0.0",
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name: "test",
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
					ContainerID: "testID",
					Image:       "testImage",
					ImageID:     "testImageID",
				},
			},
		},
	}
	pod2 := v1.Pod{
		TypeMeta: v12.TypeMeta{},
		ObjectMeta: v12.ObjectMeta{
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
		Spec: v1.PodSpec{},
		Status: v1.PodStatus{
			PodIP:  "127.0.0.2",
			HostIP: "10.0.0.0",
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name: "fake",
					State: v1.ContainerState{
						Running: &v1.ContainerStateRunning{},
					},
					ContainerID: "fakeID",
					Image:       "fakeImage",
					ImageID:     "fakeImageID",
				},
			},
		},
	}

	podList := v1.PodList{
		TypeMeta: v12.TypeMeta{},
		ListMeta: v12.ListMeta{},
		Items:    []v1.Pod{pod1, pod2},
	}

	client := fakeHttpClient(podList)
	k, _ := kubernetes.NewKubeletWithClient(&client)
	return k
}

func fakeHttpClient(pods v1.PodList) http.HttpClient {
	return &FakeHttpClient{pods: pods}
}

type FakeHttpClient struct {
	pods v1.PodList
}

func (k *FakeHttpClient) Get(path string) ([]byte, error) {
	return json.Marshal(k.pods)
}

func items() map[string]DiscoveredItem {
	items := map[string]DiscoveredItem{
		"test": {
			Variables: map[string]string{
				cluster:                   "",
				node:                      "",
				nodeIP:                    "10.0.0.0",
				namespace:                 "test",
				podName:                   "test",
				ip:                        "127.0.0.1",
				name:                      "test",
				id:                        "testID",
				image:                     "testImage",
				labelPrefix + "team":      "caos",
				annotationPrefix + "test": "test",
			},
			MetricAnnotations: map[string]string{
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
			Variables: map[string]string{
				cluster:                   "",
				node:                      "",
				nodeIP:                    "10.0.0.0",
				namespace:                 "fake",
				podName:                   "fake",
				ip:                        "127.0.0.2",
				name:                      "fake",
				id:                        "fakeID",
				image:                     "fakeImage",
				labelPrefix + "team":      "caos",
				annotationPrefix + "fake": "fake",
			},
			MetricAnnotations: map[string]string{
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
