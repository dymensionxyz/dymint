package client

import (
	"fmt"

	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
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

// walletAddressString returns the walletAddress in string format, using client pubkey.
func (c *Client) walletAddressString(wAddr *walletAddress) (string, error) {
	path := walletAddressPath(wAddr)

	addr, err := libkaspawallet.Address(c.params, []string{c.publicKey.String()}, 1, path, false)
	if err != nil {
		return "", err
	}

	return addr.String(), nil
}
