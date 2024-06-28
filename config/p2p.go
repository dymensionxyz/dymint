package config

import (
	"fmt"
	"time"
)

// P2PConfig stores configuration related to peer-to-peer networking.
type P2PConfig struct {
	// Listening address for P2P connections
	ListenAddress string `mapstructure:"p2p_listen_address"`
	// List of nodes used for P2P bootstrapping
	BootstrapNodes string `mapstructure:"p2p_bootstrap_nodes"`
	// List of nodes persistent P2P nodes
	PersistentNodes string `mapstructure:"p2p_persistent_nodes"`
	// Size of the Gossipsub router cache
	GossipSubCacheSize int `mapstructure:"p2p_gossip_cache_size"`
	// Time interval a node tries to bootstrap again, in case no nodes connected
	BootstrapRetryTime time.Duration `mapstructure:"p2p_bootstrap_retry_time"`
	// Time interval a node checks for missing blocks and tries to retrieve them from other peers using block-sync
	BlockSyncRetrieveRetryTime time.Duration `mapstructure:"p2p_blocksync_rtrv_retry_time"`
	// Time interval used by block-sync to check for missing cids in the DHT to readvertise
	BlockSyncAdvRetryTime time.Duration `mapstructure:"p2p_blocksync_adv_retry_time"`
	// Param used to enable the advertisement of the node to be part of the P2P network in the DHT
	AdvertisingEnabled bool `mapstructure:"p2p_advertising_enabled"`
}

// Validate P2PConfig
func (c P2PConfig) Validate() error {
	if c.GossipSubCacheSize < 0 {
		return fmt.Errorf("gossipsub cache size cannot be negative")
	}
	if c.BootstrapRetryTime < 0 {
		return fmt.Errorf("bootstrap time must be positive")
	}
	if c.BlockSyncRetrieveRetryTime < 0 {
		return fmt.Errorf("blocksync retrieve time must be positive")
	}
	if c.BlockSyncAdvRetryTime < 0 {
		return fmt.Errorf("blocksync adv time must be positive")
	}
	return nil
}
