package config

type RPCConfig struct {
	ListenAddress string

	CORSAllowedOrigins []string
	CORSAllowedMethods []string
	CORSAllowedHeaders []string

	MaxOpenConnections int

	TLSCertFile string `mapstructure:"tls-cert-file"`

	TLSKeyFile string `mapstructure:"tls-key-file"`
}
