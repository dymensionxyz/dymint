package config

import (
	"fmt"
	"time"
)


type P2PConfig struct {
	
	ListenAddress string `mapstructure:"p2p_listen_address"`
	
	BootstrapNodes string `mapstructure:"p2p_bootstrap_nodes"`
	
	PersistentNodes string `mapstructure:"p2p_persistent_nodes"`
	
	GossipSubCacheSize int `mapstructure:"p2p_gossip_cache_size"`
	
	BootstrapRetryTime time.Duration `mapstructure:"p2p_bootstrap_retry_time"`
	
	BlockSyncEnabled bool `mapstructure:"p2p_blocksync_enabled"`
	
	BlockSyncRequestIntervalTime time.Duration `mapstructure:"p2p_blocksync_block_request_interval"`
	
	AdvertisingEnabled bool `mapstructure:"p2p_advertising_enabled"`
}


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
