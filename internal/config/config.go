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
	ssl        = "ssl"
	envPrefix  = "NRIA"
)

var _ = flag.String(namespaces, "", "(optional, default '') Comma separated list of namespaces to discover pods on")
var _ = flag.Bool(insecure, false, `(optional, default false, deprecated) Use insecure (non-ssl) connection.
For backwards compatibility this flag takes precedence over 'ssl')`)
var _ = flag.Bool(ssl, false, "(optional, default false) Use secure (ssl) connection")
var _ = flag.Int(port, 10255, "(optional, default 10255) Port used to connect to the kubelet")

// Config defined the currently accepted configuration parameters of the Discoverer
type Config struct {
	Namespaces []string
	Port       int
	SSL        bool
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
	_ = v.BindPFlag(ssl, flag.Lookup(ssl))
	_ = v.BindPFlag(insecure, flag.Lookup(insecure))

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	// keep backwards compat
	var useSSL = v.GetBool(ssl)
	if v.IsSet(insecure) {
		useSSL = !v.GetBool(insecure)
	}

	return Config{
		Namespaces: splitStrings(v.GetString(namespaces)),
		Port:       v.GetInt(port),
		SSL:        useSSL,
	}
}
