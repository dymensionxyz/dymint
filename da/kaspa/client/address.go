package client

import (
	"fmt"
)

type walletAddress struct {
	index         uint32
	cosignerIndex uint32
	keyChain      uint8
}

func walletAddressPath(wAddr *walletAddress) string {
	return fmt.Sprintf("m/%d/%d", wAddr.keyChain, wAddr.index)
}

func walletAddressesContain(addresses []*walletAddress, contain *walletAddress) bool {
	for _, address := range addresses {
		if *address == *contain {
			return true
		}
	}

	return false
}
