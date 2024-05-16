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
	// Size of the Gossipsub router cache
	GossipedBlocksCacheSize int `mapstructure:"p2p_gossiped_blocks_cache_size"`
	// Time interval a node tries to bootstrap again, in case no nodes connected
	BootstrapRetryTime time.Duration `mapstructure:"p2p_bootstrap_retry_time"`
	// Param used to enable the advertisement of the node to be part of the P2P network in the DHT
	AdvertisingEnabled bool `mapstructure:"p2p_advertising_enabled"`
}

// Validate P2PConfig
func (c P2PConfig) Validate() error {
	if c.GossipedBlocksCacheSize < 0 {
		return fmt.Errorf("gossipsub cache size cannot be negative")
	}
	if c.BootstrapRetryTime <= 0 {
		return fmt.Errorf("bootstrap time must be positive")
	}
	return nil
}
