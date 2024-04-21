package conv

import (
	"errors"

	tmcfg "github.com/tendermint/tendermint/config"

	"github.com/dymensionxyz/dymint/config"
)

// GetNodeConfig translates Tendermint's configuration into Dymint configuration.
//
// This method only translates configuration, and doesn't verify it. If some option is missing in Tendermint's
// config, it's skipped during translation.
func GetNodeConfig(nodeConf *config.NodeConfig, tmConf *tmcfg.Config) error {
	if tmConf == nil {
		return errors.New("tendermint config is nil but required to populate Dymint config")
	}
	nodeConf.RootDir = tmConf.RootDir
	nodeConf.DBPath = tmConf.DBPath
	if tmConf.P2P != nil {
		nodeConf.P2P.ListenAddress = tmConf.P2P.ListenAddress
		nodeConf.P2P.Seeds = tmConf.P2P.Seeds
	}
	if tmConf.RPC != nil {
		nodeConf.RPC.ListenAddress = tmConf.RPC.ListenAddress
		nodeConf.RPC.CORSAllowedOrigins = tmConf.RPC.CORSAllowedOrigins
		nodeConf.RPC.CORSAllowedMethods = tmConf.RPC.CORSAllowedMethods
		nodeConf.RPC.CORSAllowedHeaders = tmConf.RPC.CORSAllowedHeaders
		nodeConf.RPC.MaxOpenConnections = tmConf.RPC.MaxOpenConnections
		nodeConf.RPC.TLSCertFile = tmConf.RPC.TLSCertFile
		nodeConf.RPC.TLSKeyFile = tmConf.RPC.TLSKeyFile
	}

	err := TranslateAddresses(nodeConf)
	if err != nil {
		return err
	}

	return nil
}
