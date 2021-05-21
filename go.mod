module github.com/newrelic/nri-discovery-kubernetes

go 1.13

require (
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/newrelic/nri-kubernetes v1.26.10-0.20210521082429-20176d06dad8
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.2
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
)
