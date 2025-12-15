# DA Instructions

## Requirements

- Access to an Aptos RPC node (e.g., https://fullnode.testnet.aptoslabs.com/v1 for testnet, https://fullnode.mainnet.aptoslabs.com/v1 for mainnet)
- An Aptos account with funds. Faucet for testnet: https://aptoslabs.com/testnet-faucet

## Dymint.toml Configuration

- Example using key file path (recommended for production):

```shell
da_layer = ['aptos']
da_config = ['{"network":"testnet","keypath_env":"APT_KEY_PATH"}']
```

Set the environment variable to point to your key file:
```shell
export APT_KEY_PATH=/path/to/private_key.txt
```

- Example using mnemonic:

```shell
da_layer = ['aptos']
da_config = ['{"network":"testnet","mnemonic_env":"APT_MNEMONIC"}']
```

Set the environment variable with your mnemonic:
```shell
export APT_MNEMONIC="your twelve word mnemonic phrase here..."
```

where:

- network: Aptos network to connect to ("testnet", "mainnet", "devnet")
- keypath_env: Environment variable name containing the path to the private key file
- mnemonic_env: Environment variable name containing the BIP39 mnemonic phrase
- retry_attempts: (Optional) Number of retry attempts for failed operations (default: 5)
- retry_delay: (Optional) Delay between retry attempts (default: 3s)

Note: Either `keypath_env` or `mnemonic_env` must be configured. If both are set, `keypath_env` takes precedence.
