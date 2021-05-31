package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/newrelic/nri-discovery-kubernetes/internal/config"
	"github.com/newrelic/nri-discovery-kubernetes/internal/discovery"
	"github.com/newrelic/nri-discovery-kubernetes/internal/kubernetes"
)

// Version of the integration
var integrationVersion = "dev"

func main() {
	config := config.NewConfig(integrationVersion)

	timeout := time.Duration(config.Timeout) * time.Millisecond
	kubelet, err := kubernetes.NewKubelet(config.Host, config.Port, config.TLS, config.IsAutoConfig(), timeout)
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
