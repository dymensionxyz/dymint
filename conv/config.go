package conv

import (
	"errors"

	tmcfg "github.com/tendermint/tendermint/config"

	"github.com/dymensionxyz/dymint/config"
)





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
	
	nodeConf.MempoolConfig = *tmConf.Mempool

	return nil
}
