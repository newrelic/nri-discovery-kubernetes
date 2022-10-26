package kubelet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func getPod(phase corev1.PodPhase, containerStatus ...corev1.ContainerStatus) corev1.Pod {
	return corev1.Pod{
		Status: corev1.PodStatus{
			Phase:             phase,
			ContainerStatuses: containerStatus,
		},
	}
}

func buildContainerStatus(containerName string, containerState corev1.ContainerState) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name:                 containerName,
		State:                containerState,
		LastTerminationState: corev1.ContainerState{},
		Ready:                true,
		RestartCount:         1,
		Image:                "k8s.gcr.io/kube-scheduler:v1.18.2",
		ImageID:              "docker-pullable://k8s.gcr.io/kube-scheduler@sha256:69f90a33b64c99e4c78e3cae36b0c767729b5a54203aa35524b1033708d1b482",
		ContainerID:          "docker://fd5fd1918be39db9992067f87f4daa755c83adbec63aece69879fb29d45514a0",
	}
}

func buildContainerStatusWaiting(containerName string) corev1.ContainerStatus {
	return buildContainerStatus(containerName, corev1.ContainerState{
		Waiting:    &corev1.ContainerStateWaiting{},
		Running:    nil,
		Terminated: nil,
	})
}

func buildContainerStatusRunning(containerName string) corev1.ContainerStatus {
	return buildContainerStatus(containerName, corev1.ContainerState{
		Waiting:    nil,
		Running:    &corev1.ContainerStateRunning{},
		Terminated: nil,
	})
}

func buildContainerStatusTerminated(containerName string) corev1.ContainerStatus {
	return buildContainerStatus(containerName, corev1.ContainerState{
		Waiting:    nil,
		Running:    nil,
		Terminated: &corev1.ContainerStateTerminated{},
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
		pods               []corev1.Pod
		expectedContainers []ContainerInfo
	}{
		{
			testName: "Pods:1Running/Containers:2Running",
			pods: []corev1.Pod{
				getPod(
					corev1.PodRunning,
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
			pods: []corev1.Pod{
				getPod(
					corev1.PodRunning,
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
			pods: []corev1.Pod{
				getPod(
					corev1.PodRunning,
					buildContainerStatusWaiting("test1"),
					buildContainerStatusTerminated("test2")),
			},
			expectedContainers: nil,
		},
		{
			testName: "Pods:1Pending,1Succeeded,1Failed,1Unknown/Containers:4Running",
			pods: []corev1.Pod{
				getPod(
					corev1.PodPending,
					buildContainerStatusRunning("test1")),
				getPod(
					corev1.PodSucceeded,
					buildContainerStatusRunning("test2")),
				getPod(
					corev1.PodFailed,
					buildContainerStatusRunning("test3")),
				getPod(
					corev1.PodUnknown,
					buildContainerStatusRunning("test4")),
			},
			expectedContainers: nil,
		},
		{
			testName: "Pods:2Running,1Succeeded/Containers:3Running,1Waiting,1Terminated",
			pods: []corev1.Pod{
				getPod(
					corev1.PodRunning,
					buildContainerStatusRunning("test1"),
					buildContainerStatusRunning("test2")),
				getPod(
					corev1.PodRunning,
					buildContainerStatusRunning("test3"),
					buildContainerStatusWaiting("test4")),
				getPod(
					corev1.PodSucceeded,
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
