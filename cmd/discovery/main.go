package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/newrelic/nri-discovery-kubernetes/internal/discovery"
	"github.com/newrelic/nri-discovery-kubernetes/internal/http"
	kubelet "github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Version of the integration.
var integrationVersion = "dev"

const (
	exitKubernetesConfigurationReadError = iota + 1
	exitNoConnectionToKubelet
	exitJSONMarchallError
	exitKubernetesConfigurationBuildError
	exitKubernetesClientBuildError
	exitKubeletClientBuildError
)

func main() {
	config, err := config.NewConfig(integrationVersion)
	if err != nil {
		log.Printf("failed read the configuration: %s ", err)
		os.Exit(exitKubernetesConfigurationReadError)
	}

	k8sConfig, err := getK8sConfig(config)
	if err != nil {
		log.Printf("setting kubernetes configuration: %s", err)
		os.Exit(exitKubernetesConfigurationBuildError)
	}

	k8s, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		log.Printf("building kubernetes client: %s", err)
		os.Exit(exitKubernetesClientBuildError)
	}

	connector := http.DefaultConnector(k8s, config, k8sConfig, log.New())

	httpClient, err := http.NewClient(connector, http.WithMaxRetries(config.Retries))
	if err != nil {
		log.Printf("building kubelet client: %s", err)
		os.Exit(exitKubeletClientBuildError)
	}

	kube := kubelet.New(httpClient, config)
	discoverer := discovery.NewDiscoverer(config.Namespaces, kube, config.DiscoverServices)

	// If discovering services, initialize and set the service discoverer
	if config.DiscoverServices {
		serviceDiscoverer := kubelet.NewServiceDiscoverer(k8s, config)
		discoverer.SetServiceDiscoverer(serviceDiscoverer)
	}

	output, err := discoverer.Run()
	if err != nil {
		log.Printf("failed to connect to Kubernetes: %s", err)
		os.Exit(exitNoConnectionToKubelet)
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		log.Printf("failed to marshal result to Json: %s", err)
		os.Exit(exitJSONMarchallError)
	}
	fmt.Println(string(bytes))
}

func getK8sConfig(c *config.Config) (*rest.Config, error) {
	inclusterConfig, err := rest.InClusterConfig()
	if err == nil {
		return inclusterConfig, nil
	}
	log.Warnf("collecting in cluster config: %v", err)

	kubeconf := c.KubeConfigFile
	if kubeconf == "" {
		kubeconf = path.Join(homedir.HomeDir(), ".kube", "config")
	}

	inclusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeconf)
	if err != nil {
		return nil, fmt.Errorf("could not load local kube config: %w", err)
	}

	log.Warnf("using local kube config: %q", kubeconf)

	return inclusterConfig, nil
}
