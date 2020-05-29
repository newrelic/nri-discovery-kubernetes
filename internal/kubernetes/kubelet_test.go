package kubernetes

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func pointerToBool(b bool) *bool {
	return &b
}

func getPod(phase v1.PodPhase, containerStatus ...v1.ContainerStatus) v1.Pod {
	return v1.Pod{
		Status: v1.PodStatus{
			Phase:             phase,
			ContainerStatuses: containerStatus,
		},
	}
}

func buildContainerStatus(containerName string, containerState v1.ContainerState) v1.ContainerStatus {
	return v1.ContainerStatus{
		Name:                 containerName,
		State:                containerState,
		LastTerminationState: v1.ContainerState{},
		Ready:                true,
		RestartCount:         1,
		Image:                "k8s.gcr.io/kube-scheduler:v1.18.2",
		ImageID:              "docker-pullable://k8s.gcr.io/kube-scheduler@sha256:69f90a33b64c99e4c78e3cae36b0c767729b5a54203aa35524b1033708d1b482",
		ContainerID:          "docker://fd5fd1918be39db9992067f87f4daa755c83adbec63aece69879fb29d45514a0",
		Started:              pointerToBool(true),
	}
}

func buildContainerStatusWaiting(containerName string) v1.ContainerStatus {
	return buildContainerStatus(containerName, v1.ContainerState{
		Waiting:    &v1.ContainerStateWaiting{},
		Running:    nil,
		Terminated: nil,
	})
}

func buildContainerStatusRunning(containerName string) v1.ContainerStatus {
	return buildContainerStatus(containerName, v1.ContainerState{
		Waiting:    nil,
		Running:    &v1.ContainerStateRunning{},
		Terminated: nil,
	})
}

func buildContainerStatusTerminated(containerName string) v1.ContainerStatus {
	return buildContainerStatus(containerName, v1.ContainerState{
		Waiting:    nil,
		Running:    nil,
		Terminated: &v1.ContainerStateTerminated{},
	})
}

func buildExpectedContainerInfo(containerName string) ContainerInfo {
	return ContainerInfo{
		Name:           containerName,
		ID:             "docker://fd5fd1918be39db9992067f87f4daa755c83adbec63aece69879fb29d45514a0",
		Image:          "k8s.gcr.io/kube-scheduler:v1.18.2",
		ImageID:        "docker-pullable://k8s.gcr.io/kube-scheduler@sha256:69f90a33b64c99e4c78e3cae36b0c767729b5a54203aa35524b1033708d1b482",
		Ports:          PortsMap{},
		PodIP:          "",
		PodLabels:      nil,
		PodAnnotations: nil,
		PodName:        "",
		NodeName:       "testNode",
		NodeIP:         "",
		Namespace:      "",
		Cluster:        "testCluster",
	}
}

func TestGetContainers(t *testing.T) {
	testCases := []struct {
		testName           string
		pods               []v1.Pod
		expectedContainers []ContainerInfo
	}{
		{
			testName: "Pods:1Running/Containers:2Running",
			pods: []v1.Pod{
				getPod(
					v1.PodRunning,
					buildContainerStatusRunning("test1"),
					buildContainerStatusRunning("test2")),
			},
			expectedContainers: []ContainerInfo{
				buildExpectedContainerInfo("test1"),
				buildExpectedContainerInfo("test2"),
			},
		},
		{
			testName: "Pods:1Running/Containers:1Running,1Waiting,1Terminated",
			pods: []v1.Pod{
				getPod(
					v1.PodRunning,
					buildContainerStatusRunning("test1"),
					buildContainerStatusWaiting("test2"),
					buildContainerStatusTerminated("test3")),
			},
			expectedContainers: []ContainerInfo{
				buildExpectedContainerInfo("test1"),
			},
		},
		{
			testName: "Pods:1Running/Containers:1Waiting,1Terminated",
			pods: []v1.Pod{
				getPod(
					v1.PodRunning,
					buildContainerStatusWaiting("test1"),
					buildContainerStatusTerminated("test2")),
			},
			expectedContainers: nil,
		},
		{
			testName: "Pods:1Pending,1Succeeded,1Failed,1Unknown/Containers:4Running",
			pods: []v1.Pod{
				getPod(
					v1.PodPending,
					buildContainerStatusRunning("test1")),
				getPod(
					v1.PodSucceeded,
					buildContainerStatusRunning("test2")),
				getPod(
					v1.PodFailed,
					buildContainerStatusRunning("test3")),
				getPod(
					v1.PodUnknown,
					buildContainerStatusRunning("test4")),
			},
			expectedContainers: nil,
		},
		{
			testName: "Pods:2Running,1Succeeded/Containers:3Running,1Waiting,1Terminated",
			pods: []v1.Pod{
				getPod(
					v1.PodRunning,
					buildContainerStatusRunning("test1"),
					buildContainerStatusRunning("test2")),
				getPod(
					v1.PodRunning,
					buildContainerStatusRunning("test3"),
					buildContainerStatusWaiting("test4")),
				getPod(
					v1.PodSucceeded,
					buildContainerStatusTerminated("test5")),
			},
			expectedContainers: []ContainerInfo{
				buildExpectedContainerInfo("test1"),
				buildExpectedContainerInfo("test2"),
				buildExpectedContainerInfo("test3"),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.testName, func(t *testing.T) {
			actualContainers := getContainers("testCluster", "testNode", testCase.pods)
			assert.Equal(t, testCase.expectedContainers, actualContainers)
		})
	}
}
