package celestia

import (
	"bytes"
	"fmt"
	"math"
)

const (
	// NamespaceVersionZero is the first namespace version.
	NamespaceVersionZero = uint8(0)

	// NamespaceIDSize is the size of a namespace ID in bytes.
	NamespaceIDSize = 28

	// NamespaceVersionMax is the max namespace version.
	NamespaceVersionMax = math.MaxUint8

	// NamespaceZeroPrefixSize is the number of `0` bytes that are prefixed to
	// namespace IDs for version 0.
	NamespaceVersionZeroPrefixSize = 18

	// NamespaceVersionZeroIDSize is the number of bytes available for
	// user-specified namespace ID in a namespace ID for version 0.
	NamespaceVersionZeroIDSize = NamespaceIDSize - NamespaceVersionZeroPrefixSize
)

// NamespaceVersionZeroPrefix is the prefix of a namespace ID for version 0.
var NamespaceVersionZeroPrefix = bytes.Repeat([]byte{0}, NamespaceVersionZeroPrefixSize)

type Namespace struct {
	Version uint8
	ID      []byte
}

// New returns a new namespace with the provided version and id.
func New(version uint8, id []byte) (Namespace, error) {
	err := validateVersion(version)
	if err != nil {
		return Namespace{}, err
	}

	err = validateID(version, id)
	if err != nil {
		return Namespace{}, err
	}

	return Namespace{
		Version: version,
		ID:      id,
	}, nil
}

// validateVersion returns an error if the version is not supported.
func validateVersion(version uint8) error {
	if version != NamespaceVersionZero && version != NamespaceVersionMax {
		return fmt.Errorf("unsupported namespace version %v", version)
	}
	return nil
}

// validateID returns an error if the provided id does not meet the requirements
// for the provided version.
func validateID(version uint8, id []byte) error {
	if len(id) != NamespaceIDSize {
		return fmt.Errorf("unsupported namespace id length: id %v must be %v bytes but it was %v bytes", id, NamespaceIDSize, len(id))
	}

	if version == NamespaceVersionZero && !bytes.HasPrefix(id, NamespaceVersionZeroPrefix) {
		return fmt.Errorf("unsupported namespace id with version %v. ID %v must start with %v leading zeros", version, id, len(NamespaceVersionZeroPrefix))
	}
	return nil
}

// Bytes returns this namespace as a byte slice.
func (n Namespace) Bytes() []byte {
	return append([]byte{n.Version}, n.ID...)
}
