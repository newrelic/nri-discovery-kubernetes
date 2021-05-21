package config

import (
	"strings"

	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	namespaces = "namespaces"
	port       = "port"
	Host       = "host"
	insecure   = "insecure"
	timeout    = "timeout"
	tls        = "tls"
	envPrefix  = "NRIA"

	DefaultHost = "localhost"
	DefaultPort = 10255
)

var (
	_ = flag.String(namespaces, "", "(optional, default '') Comma separated list of namespaces to discover pods on")
	_ = flag.Bool(insecure, false, `(optional, default false, deprecated) Use insecure (non-ssl) connection.
For backwards compatibility this flag takes precedence over 'tls')`)
)
var (
	_ = flag.Int(timeout, 5000, "(optional, default 5000) timeout in ms")
	_ = flag.Bool(tls, false, "(optional, default false) Use secure (tls) connection")
	_ = flag.Int(port, DefaultPort, "(optional, default 10255) Port used to connect to the kubelet")
	_ = flag.String(Host, DefaultHost, "(optional, default "+DefaultHost+") Host used to connect to the kubelet")
)

// Config defined the currently accepted configuration parameters of the Discoverer
type Config struct {
	Namespaces []string
	Port       int
	Host       string
	TLS        bool
	Timeout    int
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

// IsAutoConfig returns true if no config parameter was provided as cmd line arg.
func (c *Config) IsAutoConfig() bool {
	return !IsFlagPassed(Host) && !IsFlagPassed(port) && !IsFlagPassed(tls)
}

func NewConfig(version string) Config {
	flag.Parse()

	v := viper.New()
	_ = v.BindPFlag(namespaces, flag.Lookup(namespaces))
	_ = v.BindPFlag(port, flag.Lookup(port))
	_ = v.BindPFlag(Host, flag.Lookup(Host))
	_ = v.BindPFlag(tls, flag.Lookup(tls))
	_ = v.BindPFlag(insecure, flag.Lookup(insecure))
	_ = v.BindPFlag(timeout, flag.Lookup(timeout))

	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()

	// keep backwards compat
	useTLS := v.GetBool(tls)
	if v.IsSet(insecure) {
		useTLS = !v.GetBool(insecure)
	}

	return Config{
		Namespaces: splitStrings(v.GetString(namespaces)),
		Port:       v.GetInt(port),
		Host:       v.GetString(Host),
		TLS:        useTLS,
		Timeout:    v.GetInt(timeout),
	}
}
