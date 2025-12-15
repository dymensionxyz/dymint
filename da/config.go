package da

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/cosmos/go-bip39"
	uretry "github.com/dymensionxyz/dymint/utils/retry"
)

const (
	// DefaultRetryDelay is the default delay between retry attempts
	DefaultRetryDelay = 3 * time.Second
	// DefaultRetryAttempts is the default number of retry attempts
	DefaultRetryAttempts = 5
	// DefaultTimeout is the default request timeout
	DefaultTimeout = 30 * time.Second
)

// DefaultSubmitBackoff is the default backoff configuration for DA submissions
var DefaultSubmitBackoff = uretry.NewBackoffConfig(
	uretry.WithInitialDelay(time.Second*6),
	uretry.WithMaxDelay(time.Second*6),
)

// KeyConfig contains key/mnemonic configuration for DA clients.
// Supports three methods of loading credentials (in priority order):
// 1. Mnemonic phrase directly in config (mnemonic)
// 2. Mnemonic phrase from file (mnemonic_path)
// 3. Private key from JSON file with format {"private_key": "..."} (keypath)
// At least one must be configured. This struct should be embedded
// separately from BaseConfig as not all DAs require key configuration.
type KeyConfig struct {
	// Mnemonic is the mnemonic phrase directly in config
	Mnemonic string `json:"mnemonic,omitempty"`
	// MnemonicPath is the path to a file containing the mnemonic phrase
	MnemonicPath string `json:"mnemonic_path,omitempty"`
	// KeyPath is the path to a JSON key file with format {"private_key": "..."}
	KeyPath string `json:"keypath,omitempty"`
}

// keyFile represents the JSON structure for private key files
type keyFile struct {
	PrivateKey string `json:"private_key"`
}

// GetMnemonic returns the mnemonic from the configured source.
// Priority: Mnemonic (direct) > MnemonicPath (from file)
// Returns empty string if neither is configured or available.
func (k *KeyConfig) GetMnemonic() (string, error) {
	// Direct mnemonic takes priority
	if k.Mnemonic != "" {
		return k.Mnemonic, nil
	}
	// Then try mnemonic file
	if k.MnemonicPath != "" {
		data, err := os.ReadFile(k.MnemonicPath)
		if err != nil {
			return "", fmt.Errorf("read mnemonic file: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	return "", nil
}

// GetPrivateKey returns the private key from the configured JSON key file.
// The key file must have format {"private_key": "..."}
// Returns empty string if not configured.
func (k *KeyConfig) GetPrivateKey() (string, error) {
	if k.KeyPath == "" {
		return "", nil
	}

	data, err := os.ReadFile(k.KeyPath)
	if err != nil {
		return "", fmt.Errorf("read key file: %w", err)
	}

	var kf keyFile
	if err := json.Unmarshal(data, &kf); err != nil {
		return "", fmt.Errorf("parse key file JSON: %w", err)
	}

	if kf.PrivateKey == "" {
		return "", fmt.Errorf("key file missing 'private_key' field")
	}

	return kf.PrivateKey, nil
}

// Validate checks that at least one key loading method is configured.
// Returns an error if no method is configured.
func (k *KeyConfig) Validate() error {
	if k.Mnemonic != "" || k.MnemonicPath != "" || k.KeyPath != "" {
		return nil
	}
	return fmt.Errorf("no key configuration provided: set one of mnemonic, mnemonic_path, or keypath")
}

// BaseConfig contains common configuration fields for all DA clients.
// Embed this struct in DA-specific configs to get standard retry/timeout behavior.
type BaseConfig struct {
	Backoff       uretry.BackoffConfig `json:"backoff,omitempty"`
	RetryAttempts *int                 `json:"retry_attempts,omitempty"`
	RetryDelay    time.Duration        `json:"retry_delay,omitempty"`
	Timeout       time.Duration        `json:"timeout,omitempty"`
}

// SetDefaults sets default values for unset fields
func (c *BaseConfig) SetDefaults() {
	if c.RetryDelay == 0 {
		c.RetryDelay = DefaultRetryDelay
	}
	if c.Backoff == (uretry.BackoffConfig{}) {
		c.Backoff = DefaultSubmitBackoff
	}
	if c.RetryAttempts == nil {
		attempts := DefaultRetryAttempts
		c.RetryAttempts = &attempts
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	}
}

// GetRetryAttempts returns retry attempts with a safe default
func (c *BaseConfig) GetRetryAttempts() int {
	if c.RetryAttempts == nil {
		return DefaultRetryAttempts
	}
	return *c.RetryAttempts
}

// DeriveECDSAKeyFromMnemonic derives an ECDSA private key from a BIP39 mnemonic
// using the specified BIP44 derivation path. This is used for Ethereum-compatible
// chains (ETH, BNB) that use secp256k1 keys.
// Example path: "m/44'/60'/0'/0/0" for Ethereum
func DeriveECDSAKeyFromMnemonic(mnemonic, derivationPath string) (*ecdsa.PrivateKey, error) {
	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("invalid mnemonic")
	}

	// Generate seed from mnemonic (no passphrase)
	seed := bip39.NewSeed(mnemonic, "")

	// Create master key from seed
	masterKey, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("create master key: %w", err)
	}

	// Parse and derive the path
	path, err := ParseDerivationPath(derivationPath)
	if err != nil {
		return nil, fmt.Errorf("parse derivation path: %w", err)
	}

	key := masterKey
	for _, n := range path {
		key, err = key.Derive(n)
		if err != nil {
			return nil, fmt.Errorf("derive key: %w", err)
		}
	}

	// Get the ECDSA private key
	privateKey, err := key.ECPrivKey()
	if err != nil {
		return nil, fmt.Errorf("get private key: %w", err)
	}

	return privateKey.ToECDSA(), nil
}

// ParseDerivationPath parses a BIP44 derivation path string into uint32 components.
// Supports hardened derivation indicated by ' suffix.
// Example: "m/44'/60'/0'/0/0" -> []uint32{0x8000002C, 0x8000003C, 0x80000000, 0, 0}
func ParseDerivationPath(path string) ([]uint32, error) {
	// Remove "m/" prefix if present
	path = strings.TrimPrefix(path, "m/")

	components := strings.Split(path, "/")
	result := make([]uint32, len(components))

	for i, component := range components {
		// Check if hardened (ends with ')
		hardened := strings.HasSuffix(component, "'")
		if hardened {
			component = strings.TrimSuffix(component, "'")
		}

		// Parse the number
		n, err := strconv.ParseUint(component, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid path component %s: %w", component, err)
		}

		// Apply hardened offset if needed (0x80000000)
		if hardened {
			result[i] = uint32(n) + hdkeychain.HardenedKeyStart
		} else {
			result[i] = uint32(n)
		}
	}

	return result, nil
}
