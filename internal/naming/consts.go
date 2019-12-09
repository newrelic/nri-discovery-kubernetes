package naming

type DiscoveryProperty = string

const (
	LabelPrefix      DiscoveryProperty = "label."
	AnnotationPrefix DiscoveryProperty = "annotation."
	Cluster          DiscoveryProperty = "clusterName"
	Namespace        DiscoveryProperty = "namespace"
	NodeIP           DiscoveryProperty = "nodeIP"
	NodeName         DiscoveryProperty = "nodeName"
	PodName          DiscoveryProperty = "podName"
	Image            DiscoveryProperty = "image"
	Name             DiscoveryProperty = "name"
	Id               DiscoveryProperty = "id"
	IP               DiscoveryProperty = "ip"
)
