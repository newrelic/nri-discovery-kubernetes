package discovery

type Property = string

const (
	labelPrefix      Property = "label."
	annotationPrefix Property = "annotation."
	cluster          Property = "clusterName"
	namespace        Property = "namespace"
	NodeIP           Property = "nodeIP"
	NodeName         Property = "nodeName"
	podName          Property = "podName"
	image            Property = "image"
	name             Property = "name"
	id               Property = "id"
	ip               Property = "ip"

	entityRewriteActionReplace Property = "replace"
)
