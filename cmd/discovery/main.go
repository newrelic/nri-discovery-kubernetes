package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	cfg "github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/newrelic/nri-discovery-kubernetes/internal/discovery"
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
)

var (
	// Version of the integration
	Version = "dev"
)

func main() {

	config := cfg.NewConfig(Version)

	kubelet, err := kubernetes.NewKubelet(config.Port, config.TLS)
	if err != nil {
		log.Printf("failed to get Kubernetes configuration: %s", err)
		os.Exit(1)
	}
	discoverer := discovery.NewDiscoverer(config.Namespaces, kubelet)
	output, err := discoverer.Run()
	if err != nil {
		log.Printf("failed to connect to Kubernetes: %s", err)
		os.Exit(2)
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		log.Printf("failed to marshal result to Json: %s", err)
		os.Exit(3)
	}
	fmt.Println(string(bytes))
}
