package discovery

// Property represents a discovery field.
type Property = string

const (
	labelPrefix      Property = "label."
	annotationPrefix Property = "annotation."
	cluster          Property = "clusterName"
	namespace        Property = "namespace"
	nodeIP           Property = "nodeIP"
	node             Property = "nodeName"
	podName          Property = "podName"
	image            Property = "image"
	name             Property = "name"
	id               Property = "id"
	ip               Property = "ip"
	ports            Property = "ports"

	// Service-specific properties
	serviceName      Property = "serviceName"
	serviceType      Property = "serviceType"
	clusterIP        Property = "clusterIP"
	externalIPs      Property = "externalIPs"
	serviceSelector  Property = "selector"

	entityRewriteActionReplace Property = "replace"
	entityRewriteMatch         Property = "${ip}"
	entityReplaceField         Property = "k8s:${clusterName}:${namespace}:pod:${podName}:${name}"
	serviceEntityReplaceField  Property = "k8s:${clusterName}:${namespace}:service:${serviceName}"
)
