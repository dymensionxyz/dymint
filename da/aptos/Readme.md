# DA Instructions

## Requirements

- Access to an Aptos RPC node (e.g., https://fullnode.testnet.aptoslabs.com/v1 for testnet, https://fullnode.mainnet.aptoslabs.com/v1 for mainnet)
- An Aptos account with funds. Faucet for testnet: https://aptoslabs.com/testnet-faucet

## Dymint.toml Configuration

- Example:

```shell
da_layer = ['aptos']
da_config = ['{"network_id":"testnet","keypath":"/path/to/aptos_key.json"}']
```

where:

- network_id: Aptos network to connect to ("testnet", "mainnet", "devnet").
- keypath: path to JSON file containing the private key (format: `{"private_key": "0x..."}`).
- retry_attempts: (Optional) Number of retry attempts for failed operations (default: 5).
- retry_delay: (Optional) Delay between retry attempts (default: 3s).

Note: Aptos DA only supports private key file configuration (mnemonic is not supported).
