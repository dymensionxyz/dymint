package kaspa

import (
	"fmt"

	"github.com/kaspanet/kaspad/app/appmessage"
	"github.com/kaspanet/kaspad/cmd/kaspawallet/libkaspawallet"
	"github.com/kaspanet/kaspad/util"
	"github.com/pkg/errors"
)

var keyChains = []uint8{libkaspawallet.ExternalKeychain, libkaspawallet.InternalKeychain}

func (s *Client) changeAddress(useExisting bool, fromAddresses []*walletAddress) (util.Address, *walletAddress, error) {
	var walletAddr *walletAddress
	if len(fromAddresses) != 0 && useExisting {
		walletAddr = fromAddresses[0]
	} else {
		internalIndex := uint32(0)
		if !useExisting {
			err := s.keysFile.SetLastUsedInternalIndex(s.keysFile.LastUsedInternalIndex() + 1)
			if err != nil {
				return nil, nil, err
			}

			err = s.keysFile.Save()
			if err != nil {
				return nil, nil, err
			}

			internalIndex = s.keysFile.LastUsedInternalIndex()
		}

		walletAddr = &walletAddress{
			index:         internalIndex,
			cosignerIndex: s.keysFile.CosignerIndex,
			keyChain:      libkaspawallet.InternalKeychain,
		}
	}

	path := s.walletAddressPath(walletAddr)
	address, err := libkaspawallet.Address(s.params, s.keysFile.ExtendedPublicKeys, s.keysFile.MinimumSignatures, path, s.keysFile.ECDSA)
	if err != nil {
		return nil, nil, err
	}
	return address, walletAddr, nil
}

/*func (s *Client) ShowAddresses(_ context.Context, request *pb.ShowAddressesRequest) (*pb.ShowAddressesResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isSynced() {
		return nil, errors.Errorf("wallet daemon is not synced yet, %s", s.formatSyncStateReport())
	}

	addresses := make([]string, s.keysFile.LastUsedExternalIndex())
	for i := uint32(1); i <= s.keysFile.LastUsedExternalIndex(); i++ {
		walletAddr := &walletAddress{
			index:         i,
			cosignerIndex: s.keysFile.CosignerIndex,
			keyChain:      libkaspawallet.ExternalKeychain,
		}
		path := s.walletAddressPath(walletAddr)
		address, err := libkaspawallet.Address(s.params, s.keysFile.ExtendedPublicKeys, s.keysFile.MinimumSignatures, path, s.keysFile.ECDSA)
		if err != nil {
			return nil, err
		}
		addresses[i-1] = address.String()
	}

	return &pb.ShowAddressesResponse{Address: addresses}, nil
}

func (s *Client) NewAddress(_ context.Context, request *pb.NewAddressRequest) (*pb.NewAddressResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isSynced() {
		return nil, errors.Errorf("wallet daemon is not synced yet, %s", s.formatSyncStateReport())
	}

	err := s.keysFile.SetLastUsedExternalIndex(s.keysFile.LastUsedExternalIndex() + 1)
	if err != nil {
		return nil, err
	}

	err = s.keysFile.Save()
	if err != nil {
		return nil, err
	}

	walletAddr := &walletAddress{
		index:         s.keysFile.LastUsedExternalIndex(),
		cosignerIndex: s.keysFile.CosignerIndex,
		keyChain:      libkaspawallet.ExternalKeychain,
	}
	path := s.walletAddressPath(walletAddr)
	address, err := libkaspawallet.Address(s.params, s.keysFile.ExtendedPublicKeys, s.keysFile.MinimumSignatures, path, s.keysFile.ECDSA)
	if err != nil {
		return nil, err
	}

	return &pb.NewAddressResponse{Address: address.String()}, nil
}*/

func (s *Client) walletAddressString(wAddr *walletAddress) (string, error) {
	path := s.walletAddressPath(wAddr)
	addr, err := libkaspawallet.Address(s.params, s.keysFile.ExtendedPublicKeys, s.keysFile.MinimumSignatures, path, s.keysFile.ECDSA)
	if err != nil {
		return "", err
	}

	return addr.String(), nil
}

func (s *Client) walletAddressPath(wAddr *walletAddress) string {
	if s.isMultisig() {
		return fmt.Sprintf("m/%d/%d/%d", wAddr.cosignerIndex, wAddr.keyChain, wAddr.index)
	}
	return fmt.Sprintf("m/%d/%d", wAddr.keyChain, wAddr.index)
}

func (s *Client) isMultisig() bool {
	return len(s.keysFile.ExtendedPublicKeys) > 1
}

// collectFarAddresses collects numIndexesToQueryForFarAddresses addresses
// from the last point it stopped in the previous call.
func (s *Client) collectFarAddresses() error {

	err := s.collectAddresses(s.nextSyncStartIndex, s.nextSyncStartIndex+numIndexesToQueryForFarAddresses)
	if err != nil {
		return err
	}

	s.nextSyncStartIndex += numIndexesToQueryForFarAddresses
	return nil
}

func (s *Client) collectAddresses(start, end uint32) error {
	addressSet, err := s.addressesToQuery(start, end)
	if err != nil {
		return err
	}

	getBalancesByAddressesResponse, err := s.rpcClient.GetBalancesByAddresses(addressSet.strings())
	if err != nil {
		return err
	}

	err = s.updateAddressesAndLastUsedIndexes(addressSet, getBalancesByAddressesResponse)
	if err != nil {
		return err
	}

	return nil
}

// addressesToQuery scans the addresses in the given range. Because
// each cosigner in a multisig has its own unique path for generating
// addresses it goes over all the cosigners and add their addresses
// for each key chain.
func (s *Client) addressesToQuery(start, end uint32) (walletAddressSet, error) {
	addresses := make(walletAddressSet)
	for index := start; index < end; index++ {
		for cosignerIndex := uint32(0); cosignerIndex < uint32(len(s.keysFile.ExtendedPublicKeys)); cosignerIndex++ {
			for _, keychain := range keyChains {
				address := &walletAddress{
					index:         index,
					cosignerIndex: cosignerIndex,
					keyChain:      keychain,
				}
				addressString, err := s.walletAddressString(address)
				if err != nil {
					return nil, err
				}
				addresses[addressString] = address
			}
		}
	}

	return addresses, nil
}

func (s *Client) updateAddressesAndLastUsedIndexes(requestedAddressSet walletAddressSet,
	getBalancesByAddressesResponse *appmessage.GetBalancesByAddressesResponseMessage) error {
	lastUsedExternalIndex := s.keysFile.LastUsedExternalIndex()
	lastUsedInternalIndex := s.keysFile.LastUsedInternalIndex()

	for _, entry := range getBalancesByAddressesResponse.Entries {
		walletAddress, ok := requestedAddressSet[entry.Address]
		if !ok {
			return errors.Errorf("Got result from address %s even though it wasn't requested", entry.Address)
		}

		if entry.Balance == 0 {
			continue
		}

		s.addressSet[entry.Address] = walletAddress

		if walletAddress.keyChain == libkaspawallet.ExternalKeychain {
			if walletAddress.index > lastUsedExternalIndex {
				lastUsedExternalIndex = walletAddress.index
			}
			continue
		}

		if walletAddress.index > lastUsedInternalIndex {
			lastUsedInternalIndex = walletAddress.index
		}
	}

	err := s.keysFile.SetLastUsedExternalIndex(lastUsedExternalIndex)
	if err != nil {
		return err
	}

	return s.keysFile.SetLastUsedInternalIndex(lastUsedInternalIndex)
}

func walletAddressesContain(addresses []*walletAddress, contain *walletAddress) bool {
	for _, address := range addresses {
		if *address == *contain {
			return true
		}
	}

	return false
}
