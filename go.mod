module github.com/newrelic/nri-discovery-kubernetes

go 1.13

require (
	github.com/newrelic/nri-kubernetes/v2 v2.8.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.22.0
	k8s.io/apimachinery v0.22.0
	k8s.io/client-go v0.22.0
)

replace github.com/pkg/sftp => github.com/pkg/sftp v1.13.2
