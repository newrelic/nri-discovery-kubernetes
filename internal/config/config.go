package config

import (
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	namespaces = "namespaces"
	port       = "port"
	insecure   = "insecure"
	envPrefix  = "NRIA"
)

var _ = flag.String(namespaces, "", "(optional) Comma separated list of namespaces to discover pods on")
var _ = flag.Bool(insecure, false, "(optional) Use insecure (non-ssl) connection")
var _ = flag.Int(port, 10250, "(optional) Port used to connect to the kubelet")

// Config defined the currently accepted configuration parameters of the Discoverer
type Config struct {
	Namespaces []string
	Port       int
	Insecure   bool
}

func splitStrings(str string) []string {
	if len(str) > 0 {
		return strings.Split(str, ",")
	}
	return []string{}
}

func NewConfig(version string) Config {
	flag.Parse()

	v := viper.New()
	_ = v.BindPFlag(namespaces, flag.Lookup(namespaces))
	_ = v.BindPFlag(port, flag.Lookup(port))
	_ = v.BindPFlag(insecure, flag.Lookup(insecure))

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	return Config{
		Namespaces: splitStrings(v.GetString(namespaces)),
		Port:       v.GetInt(port),
		Insecure:   v.GetBool(insecure),
	}
}
