# DA Instructions

## Requirements

- Access to Ethereum RPC node. Public RPC addresses can be found here <https://ethereum-sepolia-rpc.publicnode.com/>, but private RPCs are recommended.
- An Ethereum account with funds. Faucet for Sepolia testnet can be found here: <https://cloud.google.com/application/web3/faucet/ethereum/sepolia>

## Dymint.toml Configuration

- Example using environment variable:

```shell
da_layer = ['eth']
da_config = ['{​"endpoint":"https://ethereum-rpc.publicnode.com","gas_limit":100000,"private_key_env":"ETH_PRIVATE_KEY","chain_id":1,"api_url":"https://ethereum-beacon-api.publicnode.com"}']
```

- Example using file (recommended for production):

```shell
da_layer = ['eth']
da_config = ['{​"endpoint":"https://ethereum-rpc.publicnode.com","gas_limit":100000,"private_key_file":"/path/to/private_key.txt","chain_id":1,"api_url":"https://ethereum-beacon-api.publicnode.com"}']
```

- Example with private key directly in config (alternative, not recommended for production):

```shell
da_layer = ['eth']
da_config = ['{​"endpoint":"https://ethereum-rpc.publicnode.com","gas_limit":100000,"private_key":"0x...","chain_id":1,"api_url":"https://ethereum-beacon-api.publicnode.com"}']
```

where:

- endpoint = RPC url address for Ethereum RPC.
- private_key_env: (Optional) Environment variable name for private key (e.g., ETH_PRIVATE_KEY). Takes precedence over other methods.
- private_key_file: (Optional) Path to file containing the private key. Takes precedence over `private_key` field.
- private_key: (Optional) Private key directly in config. Used as fallback if neither `private_key_env` nor `private_key_file` is set.
- timeout (nanoseconds): used to cancel retry when fail submissions or retrievals (optional).
- chain_id: Network identifier (11155111 for Sepolia, 1 for mainnet).
- api_url: Beacon API Endpoint Link used to retrieve blobs.
- gas_limit: max gas to be used in blob txs (optional).
