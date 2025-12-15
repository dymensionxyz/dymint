package client

import (
	"fmt"
)

// walletAddress includes the params used to generate an address from same key
type walletAddress struct {
	index         uint32
	cosignerIndex uint32
	keyChain      uint8
}

// walletAddressPath is used to generate address string
func walletAddressPath(wAddr *walletAddress) string {
	return fmt.Sprintf("m/%d/%d", wAddr.keyChain, wAddr.index)
}
