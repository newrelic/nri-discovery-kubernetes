package config

import (
	"fmt"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// Host is a Kubelet host flag name.
	Host = "host"

	// DefaultHost is default address where discovery will look for Kubelet API.
	DefaultHost = "localhost"

	// DefaultPort is default port user for Kubelet API.
	DefaultPort = 10250

	FlagNamespaces     = "namespaces"
	FlagPort           = "port"
	FlagInsecure       = "insecure"
	FlagTimeout        = "timeout"
	FlagTLS            = "tls"
	FlagKubeConfigFile = "kubeconfig"
	FlagClusterName    = "cluster_name"
	FlagNodeName       = "node_name"

	envPrefix            = "NRIA"
	nodeNameEnvVar       = "NRI_KUBERNETES_NODE_NAME"
	nodeNameEnvVarLegacy = "NRK8S_NODE_NAME"
	clusterNameEnvVar    = "CLUSTER_NAME"
)

var (
	_ = flag.String(FlagNamespaces, "", "(optional, default '') Comma separated list of namespaces to discover pods on")
	_ = flag.Bool(FlagInsecure, false, `(optional, default false, deprecated) Use insecure (non-ssl) connection.
For backwards compatibility this flag takes precedence over 'tls')`)

	_ = flag.Int(FlagTimeout, 5000, "(optional, default 5000) timeout in ms")
	_ = flag.Bool(FlagTLS, false, "(optional, default false) Use secure (tls) connection")
	_ = flag.Int(FlagPort, DefaultPort, "(optional, default 10255) Port used to connect to the kubelet")
	_ = flag.String(Host, DefaultHost, "(optional, default "+DefaultHost+") Host used to connect to the kubelet")

	_ = flag.String(FlagClusterName, "", "Set cluster name")
	_ = flag.String(FlagNodeName, "", "(optional) Set node name to try to find its IP")

	_ = flag.String(FlagKubeConfigFile, "", "(optional) Kubeconfig to use to connecto to kubelet")
)

// Config defined the currently accepted configuration parameters of the Discoverer
type Config struct {
	Namespaces     []string
	Port           int
	Host           string
	TLS            bool
	Timeout        int
	KubeConfigFile string
	ClusterName    string
	NodeName       string
}

func splitStrings(str string) []string {
	if len(str) > 0 {
		return strings.Split(str, ",")
	}
	return []string{}
}

// IsFlagPassed checks if a particular command line argument was provided or not.
func IsFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// NewConfig generates Config from flags.
func NewConfig(version string) (*Config, error) {
	flag.Parse()

	v := viper.New()
	_ = v.BindPFlag(FlagNamespaces, flag.Lookup(FlagNamespaces))
	_ = v.BindPFlag(FlagPort, flag.Lookup(FlagPort))
	_ = v.BindPFlag(Host, flag.Lookup(Host))
	_ = v.BindPFlag(FlagTLS, flag.Lookup(FlagTLS))
	_ = v.BindPFlag(FlagInsecure, flag.Lookup(FlagInsecure))
	_ = v.BindPFlag(FlagTimeout, flag.Lookup(FlagTimeout))
	_ = v.BindPFlag(FlagKubeConfigFile, flag.Lookup(FlagKubeConfigFile))

	_ = v.BindPFlag(FlagClusterName, flag.Lookup(FlagClusterName))
	_ = v.BindPFlag(FlagNodeName, flag.Lookup(FlagNodeName))

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	config := Config{
		Namespaces: splitStrings(v.GetString(FlagNamespaces)),
		Port:       v.GetInt(FlagPort),
		Host:       v.GetString(Host),
		Timeout:    v.GetInt(FlagTimeout),
	}

	// To leave the variable empty as nil
	if v.IsSet(FlagKubeConfigFile) {
		config.KubeConfigFile = v.GetString(FlagKubeConfigFile)
	}

	// keep compatibility with the old env variable
	cluster, isClusterNameSet := os.LookupEnv(clusterNameEnvVar)
	if !isClusterNameSet {
		cluster = v.GetString(FlagClusterName)
	}
	if cluster == "" {
		err := fmt.Errorf("cluster name is not set")
		return &Config{}, err
	}
	config.ClusterName = cluster

	// keep backwards compatibility from when insecure was deprecated
	useTLS := v.GetBool(FlagTLS)
	if v.IsSet(FlagInsecure) {
		useTLS = !v.GetBool(FlagInsecure)
	}
	config.TLS = useTLS

	node := v.GetString(FlagNodeName)
	nodeNew, isNewNodeNameSet := os.LookupEnv(nodeNameEnvVar) // compatibility with new nri-kubernetes v3 variable
	if isNewNodeNameSet {
		node = nodeNew
	}
	nodeLegacy, isOldNodeNameSet := os.LookupEnv(nodeNameEnvVarLegacy) // keep compatibility with the old env variable
	if isOldNodeNameSet {
		node = nodeLegacy
	}

	config.NodeName = node

	return &config, nil
}
