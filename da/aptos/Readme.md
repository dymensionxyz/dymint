# DA Instructions

## Requirements

- Access to an Aptos RPC node (e.g., https://fullnode.testnet.aptoslabs.com/v1 for testnet, https://fullnode.mainnet.aptoslabs.com/v1 for mainnet)
- An Aptos account with funds. Faucet for testnet: https://aptoslabs.com/testnet-faucet

## Dymint.toml Configuration

- Example using environment variable:

```shell
da_layer = ['aptos']
da_config = ['{​"network":"testnet","pri_key_env":"APT_PRIVATE_KEY"}']
```

- Example using file (recommended for production):

```shell
da_layer = ['aptos']
da_config = ['{​"network":"testnet","pri_key_file":"/path/to/private_key.txt"}']
```

- Example with private key directly in config (alternative, not recommended for production):

```shell
da_layer = ['aptos']
da_config = ['{​"network":"testnet","pri_key":"..."}']
```

where:

- network: Aptos network to connect to ("testnet", "mainnet", "devnet")
- pri_key_env: (Optional) Environment variable name for private key (e.g., APT_PRIVATE_KEY). Takes precedence over other methods.
- pri_key_file: (Optional) Path to file containing the private key. Takes precedence over `pri_key` field.
- pri_key: (Optional) Private key directly in config. Used as fallback if neither `pri_key_env` nor `pri_key_file` is set.
- retry_attempts: (Optional) Number of retry attempts for failed operations (default: 5)
- retry_delay: (Optional) Delay between retry attempts (default: 3s)
