package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

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

	var discoverer *discovery.Discoverer

	// Multi-resource mode: --resources flag takes precedence over --discover-services.
	if len(config.Resources) > 0 {
		discoverer = discovery.NewMultiDiscoverer(config.Namespaces, config.Resources)

		if config.HasResource("services") {
			discoverer.SetServiceDiscoverer(kubelet.NewServiceDiscoverer(k8s, config))
		}
		if config.HasResource("deployments") || config.HasResource("statefulsets") || config.HasResource("daemonsets") {
			discoverer.SetWorkloadDiscoverer(kubelet.NewWorkloadDiscoverer(
				k8s, config,
				config.HasResource("deployments"),
				config.HasResource("statefulsets"),
				config.HasResource("daemonsets"),
			))
		}
		if config.HasResource("pvcs") {
			discoverer.SetPVCDiscoverer(kubelet.NewPVCDiscoverer(k8s, config))
		}
		if config.HasResource("crds") {
			crdDisc, err := kubelet.NewCRDDiscoverer(k8sConfig)
			if err != nil {
				log.Printf("building CRD discoverer: %s", err)
				os.Exit(exitKubernetesClientBuildError)
			}
			discoverer.SetCRDDiscoverer(crdDisc)
		}
		if config.HasResource("pods") {
			connector := http.DefaultConnector(k8s, config, k8sConfig, log.New())
			httpClient, err := http.NewClient(connector, http.WithMaxRetries(config.Retries))
			if err != nil {
				log.Printf("building kubelet client: %s", err)
				os.Exit(exitKubeletClientBuildError)
			}
			discoverer.SetKubelet(kubelet.New(httpClient, config))
		}
	} else if config.DiscoverServices {
		// Legacy: --discover-services
		discoverer = discovery.NewDiscoverer(config.Namespaces, nil, true)
		discoverer.SetServiceDiscoverer(kubelet.NewServiceDiscoverer(k8s, config))
	} else {
		// Legacy default: pod/container discovery via kubelet
		connector := http.DefaultConnector(k8s, config, k8sConfig, log.New())
		httpClient, err := http.NewClient(connector, http.WithMaxRetries(config.Retries))
		if err != nil {
			log.Printf("building kubelet client: %s", err)
			os.Exit(exitKubeletClientBuildError)
		}
		kube := kubelet.New(httpClient, config)
		discoverer = discovery.NewDiscoverer(config.Namespaces, kube, false)
	}

	if config.Watch {
		runWatch(discoverer, config.WatchInterval)
		return
	}

	runOnce(discoverer)
}

func runOnce(discoverer *discovery.Discoverer) {
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

// WatchOutput wraps a discovery run with a timestamp for watch-mode output.
type WatchOutput struct {
	Timestamp string           `json:"timestamp"`
	Items     discovery.Output `json:"items"`
}

func runWatch(discoverer *discovery.Discoverer, intervalSeconds int) {
	interval := time.Duration(intervalSeconds) * time.Second
	log.Printf("watch mode: polling every %s", interval)

	for {
		output, err := discoverer.Run()
		if err != nil {
			log.Printf("discovery error: %s", err)
		} else {
			wo := WatchOutput{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Items:     output,
			}
			bytes, err := json.Marshal(wo)
			if err != nil {
				log.Printf("failed to marshal watch output: %s", err)
			} else {
				fmt.Println(string(bytes))
			}
		}
		time.Sleep(interval)
	}
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
