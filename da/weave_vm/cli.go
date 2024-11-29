package weave_vm

/*
import (
	"time"

	weaveVMtypes "github.com/dymensionxyz/dymint/da/weave_vm/types"
	"github.com/urfave/cli/v2"
)

var (
	EnabledFlagName  = withFlagPrefix("enabled")
	EndpointFlagName = withFlagPrefix("endpoint")
	ChainIDFlagName  = withFlagPrefix("chain_id")
	TimeoutFlagName  = withFlagPrefix("timeout")

	PrivateKeyHexFlagName           = withFlagPrefix("private_key_hex")
	Web3SignerEndpointFlagName      = withFlagPrefix("web3_signer_endpoint")
	Web3SignerTLSCertFileFlagName   = withFlagPrefix("web3_signer_tls_cert_file")
	Web3SignerTLSKeyFileFlagName    = withFlagPrefix("web3_signer_tls_key_file")
	Web3SignerTLSCACertFileFlagName = withFlagPrefix("web3_signer_tls_ca_cert_file")
)

func withFlagPrefix(s string) string {
	return "weavevm." + s
}

func withEnvPrefix(envPrefix, s string) []string {
	return []string{envPrefix + "_WEAVE_VM_" + s}
}

// CLIFlags ... used for WeaveVM backend configuration
// category is used to group the flags in the help output (see https://cli.urfave.org/v2/examples/flags/#grouping)
func CLIFlags(envPrefix, category string) []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:     EnabledFlagName,
			Value:    false,
			Usage:    "enable WeaveVM perm storage",
			EnvVars:  withEnvPrefix(envPrefix, "ENABLED"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     EndpointFlagName,
			Usage:    "endpoint for WeaveVM chain rpc",
			EnvVars:  withEnvPrefix(envPrefix, "ENDPOINT"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     ChainIDFlagName,
			Usage:    "chain ID of WeaveVM chain",
			EnvVars:  withEnvPrefix(envPrefix, "CHAIN_ID"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     PrivateKeyHexFlagName,
			Usage:    "private key hex of WeaveVM chain",
			EnvVars:  withEnvPrefix(envPrefix, "PRIVATE_KEY_HEX"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     Web3SignerEndpointFlagName,
			Usage:    "web3signer endpoint",
			EnvVars:  withEnvPrefix(envPrefix, "WEB3_SIGNER_ENDPOINT"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     Web3SignerTLSCertFileFlagName,
			Usage:    "web3signer tls cert file",
			EnvVars:  withEnvPrefix(envPrefix, "WEB3_SIGNER_TLS_CERT_FILE"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     Web3SignerTLSKeyFileFlagName,
			Usage:    "web3signer tls key file",
			EnvVars:  withEnvPrefix(envPrefix, "WEB3_SIGNER_TLS_KEY_FILE"),
			Category: category,
		},
		&cli.StringFlag{
			Name:     Web3SignerTLSCACertFileFlagName,
			Usage:    "web3signer tls ca cert file",
			EnvVars:  withEnvPrefix(envPrefix, "WEB3_SIGNER_TLS_CERT_FILE"),
			Category: category,
		},
		&cli.DurationFlag{
			Name:     TimeoutFlagName,
			Usage:    "timeout for WeaveVM HTTP requests operations (e.g. get, put)",
			Value:    5 * time.Second,
			EnvVars:  withEnvPrefix(envPrefix, "TIMEOUT"),
			Category: category,
		},
	}
}

func ReadConfig(ctx *cli.Context) weaveVMtypes.Config {
	return weaveVMtypes.Config{
		Endpoint: ctx.String(EndpointFlagName),
		ChainID:  ctx.Int64(ChainIDFlagName),
		Enabled:  ctx.Bool(EnabledFlagName),
		Timeout:  ctx.Duration(TimeoutFlagName),

		PrivateKeyHex:           ctx.String(PrivateKeyHexFlagName),
		Web3SignerEndpoint:      ctx.String(Web3SignerEndpointFlagName),
		Web3SignerTLSCertFile:   ctx.String(Web3SignerTLSCertFileFlagName),
		Web3SignerTLSKeyFile:    ctx.String(Web3SignerTLSKeyFileFlagName),
		Web3SignerTLSCACertFile: ctx.String(Web3SignerTLSCACertFileFlagName),
	}
}
*/
