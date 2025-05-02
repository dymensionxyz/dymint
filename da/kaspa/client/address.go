package client

import (
	"fmt"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
)

type walletAddress struct {
	index         uint32
	cosignerIndex uint32
	keyChain      uint8
}

func walletAddressPath(wAddr *walletAddress) string {
	return fmt.Sprintf("m/%d/%d", wAddr.keyChain, wAddr.index)
}

func (c *Client) walletAddressString(wAddr *walletAddress) (string, error) {
	path := walletAddressPath(wAddr)
	pubKey, err := c.extendedKey.Public()
	if err != nil {
		return "", err
	}
	addr, err := libkaspawallet.Address(c.params, []string{pubKey.String()}, 1, path, false)
	if err != nil {
		return "", err
	}

	return addr.String(), nil
}
