package da

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/dymensionxyz/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test mnemonic (DO NOT USE IN PRODUCTION)
// This is a well-known test mnemonic from BIP39 test vectors
const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// Expected addresses derived from the test mnemonic using BIP44 path m/44'/60'/0'/0/0
// These can be verified using any standard Ethereum wallet (MetaMask, etc.)
const expectedEthAddress = "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"
const expectedEthPrivateKey = "1ab42cc412b618bdea3a599e3c9bae199ebf030895b039e9db1e30dafb12b727"

func TestDeriveECDSAKeyFromMnemonic(t *testing.T) {
	t.Run("valid mnemonic with ETH path", func(t *testing.T) {
		// Standard Ethereum BIP44 path
		path := "m/44'/60'/0'/0/0"

		privateKey, err := DeriveECDSAKeyFromMnemonic(testMnemonic, path)
		require.NoError(t, err)
		require.NotNil(t, privateKey)

		// Verify the derived private key matches expected
		derivedKeyHex := hex.EncodeToString(crypto.FromECDSA(privateKey))
		assert.Equal(t, expectedEthPrivateKey, derivedKeyHex)

		// Verify the derived address matches expected
		address := crypto.PubkeyToAddress(privateKey.PublicKey)
		assert.Equal(t, expectedEthAddress, address.Hex())
	})

	t.Run("invalid mnemonic", func(t *testing.T) {
		invalidMnemonic := "invalid mnemonic phrase that should not work"
		path := "m/44'/60'/0'/0/0"

		_, err := DeriveECDSAKeyFromMnemonic(invalidMnemonic, path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mnemonic")
	})

	t.Run("different derivation paths produce different keys", func(t *testing.T) {
		path1 := "m/44'/60'/0'/0/0"
		path2 := "m/44'/60'/0'/0/1"

		key1, err := DeriveECDSAKeyFromMnemonic(testMnemonic, path1)
		require.NoError(t, err)

		key2, err := DeriveECDSAKeyFromMnemonic(testMnemonic, path2)
		require.NoError(t, err)

		// Keys should be different
		assert.NotEqual(t,
			hex.EncodeToString(crypto.FromECDSA(key1)),
			hex.EncodeToString(crypto.FromECDSA(key2)),
		)
	})

	t.Run("BNB uses same path as ETH", func(t *testing.T) {
		// BNB/BSC uses the same coin type (60) as Ethereum
		ethPath := "m/44'/60'/0'/0/0"
		bnbPath := "m/44'/60'/0'/0/0" // Same path

		ethKey, err := DeriveECDSAKeyFromMnemonic(testMnemonic, ethPath)
		require.NoError(t, err)

		bnbKey, err := DeriveECDSAKeyFromMnemonic(testMnemonic, bnbPath)
		require.NoError(t, err)

		// Should produce identical keys
		assert.Equal(t,
			hex.EncodeToString(crypto.FromECDSA(ethKey)),
			hex.EncodeToString(crypto.FromECDSA(bnbKey)),
		)
	})
}

func TestParseDerivationPath(t *testing.T) {
	t.Run("standard ETH path", func(t *testing.T) {
		path, err := ParseDerivationPath("m/44'/60'/0'/0/0")
		require.NoError(t, err)
		require.Len(t, path, 5)

		// 44' = 44 + 0x80000000
		assert.Equal(t, uint32(0x8000002C), path[0])
		// 60' = 60 + 0x80000000
		assert.Equal(t, uint32(0x8000003C), path[1])
		// 0' = 0 + 0x80000000
		assert.Equal(t, uint32(0x80000000), path[2])
		// 0 (not hardened)
		assert.Equal(t, uint32(0), path[3])
		// 0 (not hardened)
		assert.Equal(t, uint32(0), path[4])
	})

	t.Run("path without m/ prefix", func(t *testing.T) {
		path, err := ParseDerivationPath("44'/60'/0'/0/0")
		require.NoError(t, err)
		require.Len(t, path, 5)
		assert.Equal(t, uint32(0x8000002C), path[0])
	})

	t.Run("invalid path component", func(t *testing.T) {
		_, err := ParseDerivationPath("m/44'/abc/0'/0/0")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid path component")
	})
}

func TestKeyConfig_GetMnemonic(t *testing.T) {
	t.Run("direct mnemonic", func(t *testing.T) {
		cfg := KeyConfig{
			Mnemonic: testMnemonic,
		}

		mnemonic, err := cfg.GetMnemonic()
		require.NoError(t, err)
		assert.Equal(t, testMnemonic, mnemonic)
	})

	t.Run("mnemonic from file", func(t *testing.T) {
		// Create temp file with mnemonic
		tmpDir := t.TempDir()
		mnemonicFile := filepath.Join(tmpDir, "mnemonic")
		err := os.WriteFile(mnemonicFile, []byte(testMnemonic+"\n"), 0600)
		require.NoError(t, err)

		cfg := KeyConfig{
			MnemonicPath: mnemonicFile,
		}

		mnemonic, err := cfg.GetMnemonic()
		require.NoError(t, err)
		assert.Equal(t, testMnemonic, mnemonic) // Should be trimmed
	})

	t.Run("direct mnemonic takes priority over file", func(t *testing.T) {
		differentMnemonic := "zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo zoo wrong"

		// Create temp file with different mnemonic
		tmpDir := t.TempDir()
		mnemonicFile := filepath.Join(tmpDir, "mnemonic")
		err := os.WriteFile(mnemonicFile, []byte(differentMnemonic), 0600)
		require.NoError(t, err)

		cfg := KeyConfig{
			Mnemonic:     testMnemonic,
			MnemonicPath: mnemonicFile,
		}

		mnemonic, err := cfg.GetMnemonic()
		require.NoError(t, err)
		assert.Equal(t, testMnemonic, mnemonic) // Direct takes priority
	})

	t.Run("no mnemonic configured", func(t *testing.T) {
		cfg := KeyConfig{}

		mnemonic, err := cfg.GetMnemonic()
		require.NoError(t, err)
		assert.Empty(t, mnemonic)
	})

	t.Run("mnemonic file not found", func(t *testing.T) {
		cfg := KeyConfig{
			MnemonicPath: "/nonexistent/path/to/mnemonic",
		}

		_, err := cfg.GetMnemonic()
		require.Error(t, err)
	})
}

func TestKeyConfig_GetPrivateKey(t *testing.T) {
	t.Run("private key from JSON file", func(t *testing.T) {
		// Create temp JSON key file
		tmpDir := t.TempDir()
		keyFile := filepath.Join(tmpDir, "key.json")
		keyJSON := `{"private_key": "` + expectedEthPrivateKey + `"}`
		err := os.WriteFile(keyFile, []byte(keyJSON), 0600)
		require.NoError(t, err)

		cfg := KeyConfig{
			KeyPath: keyFile,
		}

		privateKey, err := cfg.GetPrivateKey()
		require.NoError(t, err)
		assert.Equal(t, expectedEthPrivateKey, privateKey)
	})

	t.Run("no keypath configured", func(t *testing.T) {
		cfg := KeyConfig{}

		privateKey, err := cfg.GetPrivateKey()
		require.NoError(t, err)
		assert.Empty(t, privateKey)
	})

	t.Run("key file not found", func(t *testing.T) {
		cfg := KeyConfig{
			KeyPath: "/nonexistent/path/to/key.json",
		}

		_, err := cfg.GetPrivateKey()
		require.Error(t, err)
	})

	t.Run("invalid JSON in key file", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyFile := filepath.Join(tmpDir, "key.json")
		err := os.WriteFile(keyFile, []byte("not valid json"), 0600)
		require.NoError(t, err)

		cfg := KeyConfig{
			KeyPath: keyFile,
		}

		_, err = cfg.GetPrivateKey()
		require.Error(t, err)
	})

	t.Run("missing private_key field", func(t *testing.T) {
		tmpDir := t.TempDir()
		keyFile := filepath.Join(tmpDir, "key.json")
		err := os.WriteFile(keyFile, []byte(`{"other_field": "value"}`), 0600)
		require.NoError(t, err)

		cfg := KeyConfig{
			KeyPath: keyFile,
		}

		_, err = cfg.GetPrivateKey()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'private_key' field")
	})
}

func TestKeyConfig_Validate(t *testing.T) {
	t.Run("mnemonic configured", func(t *testing.T) {
		cfg := KeyConfig{Mnemonic: testMnemonic}
		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("mnemonic_path configured", func(t *testing.T) {
		cfg := KeyConfig{MnemonicPath: "/some/path"}
		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("keypath configured", func(t *testing.T) {
		cfg := KeyConfig{KeyPath: "/some/path"}
		err := cfg.Validate()
		require.NoError(t, err)
	})

	t.Run("no config provided", func(t *testing.T) {
		cfg := KeyConfig{}
		err := cfg.Validate()
		require.Error(t, err)
	})
}

func TestMnemonicAndKeyPathProduceSameKey(t *testing.T) {
	// This test verifies that deriving a key from mnemonic produces the same
	// result as loading the corresponding private key from a file

	t.Run("ETH: mnemonic derivation matches private key file", func(t *testing.T) {
		// Derive key from mnemonic
		derivedKey, err := DeriveECDSAKeyFromMnemonic(testMnemonic, "m/44'/60'/0'/0/0")
		require.NoError(t, err)

		// Create a key file with the expected private key
		tmpDir := t.TempDir()
		keyFile := filepath.Join(tmpDir, "key.json")
		keyJSON := `{"private_key": "` + expectedEthPrivateKey + `"}`
		err = os.WriteFile(keyFile, []byte(keyJSON), 0600)
		require.NoError(t, err)

		// Load key from file
		cfg := KeyConfig{KeyPath: keyFile}
		loadedKeyHex, err := cfg.GetPrivateKey()
		require.NoError(t, err)

		// Parse the loaded hex key
		loadedKey, err := crypto.HexToECDSA(loadedKeyHex)
		require.NoError(t, err)

		// Both should produce the same address
		derivedAddr := crypto.PubkeyToAddress(derivedKey.PublicKey)
		loadedAddr := crypto.PubkeyToAddress(loadedKey.PublicKey)
		assert.Equal(t, derivedAddr, loadedAddr)

		// Both should be the same private key
		assert.Equal(t,
			hex.EncodeToString(crypto.FromECDSA(derivedKey)),
			hex.EncodeToString(crypto.FromECDSA(loadedKey)),
		)
	})
}
