module github.com/newrelic/nri-discovery-kubernetes

go 1.13

require (
	github.com/avast/retry-go v3.0.0+incompatible
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/newrelic/nri-kubernetes/v2 v2.5.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
