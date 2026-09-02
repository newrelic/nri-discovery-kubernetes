package discovery

// Property represents a discovery field.
type Property = string

const (
	labelPrefix         Property = "label."
	annotationPrefix    Property = "annotation."
	cluster             Property = "clusterName"
	namespace           Property = "namespace"
	nodeIP              Property = "nodeIP"
	node                Property = "nodeName"
	podName             Property = "podName"
	image               Property = "image"
	name                Property = "name"
	id                  Property = "id"
	ip                  Property = "ip"
	ports               Property = "ports"
	instrumented        Property = "instrumented"
	instrumentationType Property = "instrumentationType"
	kind                Property = "kind"

	// Service-specific properties
	serviceName     Property = "serviceName"
	serviceType     Property = "serviceType"
	clusterIP       Property = "clusterIP"
	externalIPs     Property = "externalIPs"
	serviceSelector Property = "selector"

	// Workload-specific properties (Deployment / StatefulSet / DaemonSet)
	workloadName              Property = "workloadName"
	workloadReplicas          Property = "replicas"
	workloadReadyReplicas     Property = "readyReplicas"
	workloadAvailableReplicas Property = "availableReplicas"
	workloadImages            Property = "images" // comma-separated list of container images

	// PVC-specific properties
	pvcName             Property = "pvcName"
	pvcStorageClass     Property = "storageClass"
	pvcAccessModes      Property = "accessModes"
	pvcCapacity         Property = "capacity"
	pvcStatus           Property = "pvcStatus"
	pvcVolumeName       Property = "volumeName"

	// CRD-specific properties
	crdName    Property = "crdName"
	crdGroup   Property = "crdGroup"
	crdScope   Property = "crdScope"
	crdVersion Property = "crdVersion"

	entityRewriteActionReplace Property = "replace"
	entityRewriteMatch         Property = "${ip}"
	entityReplaceField         Property = "k8s:${clusterName}:${namespace}:pod:${podName}:${name}"
	serviceEntityReplaceField  Property = "k8s:${clusterName}:${namespace}:service:${serviceName}"
	workloadEntityReplaceField Property = "k8s:${clusterName}:${namespace}:${kind}:${workloadName}"
	pvcEntityReplaceField      Property = "k8s:${clusterName}:${namespace}:pvc:${pvcName}"
)
