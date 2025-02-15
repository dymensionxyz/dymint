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
	// Param used to enable block sync from p2p
	BlockSyncEnabled bool `mapstructure:"p2p_blocksync_enabled"`
	// Time interval used by a node to request missing blocks (gap between cached blocks and local height) on demand from other peers using blocksync
	BlockSyncRequestIntervalTime time.Duration `mapstructure:"p2p_blocksync_block_request_interval"`
	// Param used to enable the advertisement and discovery of other nodes for automatic connection to other peers in the P2P network
	DiscoveryEnabled bool `mapstructure:"p2p_discovery_enabled"`
}

// Validate P2PConfig
func (c P2PConfig) Validate() error {
	if c.GossipSubCacheSize < 0 {
		return fmt.Errorf("gossipsub cache size cannot be negative")
	}
	if c.BootstrapRetryTime <= 0 {
		return fmt.Errorf("bootstrap time must be positive")
	}
	if c.BlockSyncRequestIntervalTime <= 0 {
		return fmt.Errorf("blocksync retrieve time must be positive")
	}

	return nil
}
