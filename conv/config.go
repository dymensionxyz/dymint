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
	if tmConf.RPC != nil {
		nodeConf.RPC.ListenAddress = tmConf.RPC.ListenAddress
		nodeConf.RPC.CORSAllowedOrigins = tmConf.RPC.CORSAllowedOrigins
		nodeConf.RPC.CORSAllowedMethods = tmConf.RPC.CORSAllowedMethods
		nodeConf.RPC.CORSAllowedHeaders = tmConf.RPC.CORSAllowedHeaders
		nodeConf.RPC.MaxOpenConnections = tmConf.RPC.MaxOpenConnections
		nodeConf.RPC.TLSCertFile = tmConf.RPC.TLSCertFile
		nodeConf.RPC.TLSKeyFile = tmConf.RPC.TLSKeyFile
	}
	if tmConf.Mempool == nil {
		return errors.New("tendermint mempool config is nil but required to populate Dymint config")
	}
	/*
		In the above, we are copying the rpc/p2p from Tendermint's configuration to Dymint's configuration.
		This was implemented by the original rollkit authors, and they have not provided any explanation for this.

		For the mempool we simply copy the object. If we want to be more selective, we can adjust later.
	*/
	nodeConf.MempoolConfig = *tmConf.Mempool

	return nil
}
