# DA Instructions

## Requirements

- Access to Ethereum RPC node. Public RPC addresses can be found here: <https://ethereum-sepolia-rpc.publicnode.com/>, but private RPCs are recommended.
- An Ethereum account with funds. Faucet for Sepolia testnet can be found here: <https://cloud.google.com/application/web3/faucet/ethereum/sepolia>.

## Dymint.toml Configuration

- Example using key file:

```shell
da_layer = ['eth']
da_config = ['{"endpoint":"https://ethereum-rpc.publicnode.com","gas_limit":100000,"keypath":"/path/to/eth_key.json","network_id":1,"api_url":"https://ethereum-beacon-api.publicnode.com"}']
```

- Example using mnemonic directly:

```shell
da_layer = ['eth']
da_config = ['{"endpoint":"https://ethereum-rpc.publicnode.com","gas_limit":100000,"mnemonic":"your twelve word mnemonic phrase here","network_id":1,"api_url":"https://ethereum-beacon-api.publicnode.com"}']
```

- Example using mnemonic from file:

```shell
da_layer = ['eth']
da_config = ['{"endpoint":"https://ethereum-rpc.publicnode.com","gas_limit":100000,"mnemonic_path":"/path/to/mnemonic.txt","network_id":1,"api_url":"https://ethereum-beacon-api.publicnode.com"}']
```

where:

- endpoint: RPC url address for Ethereum RPC.
- keypath: path to JSON file containing the private key (format: `{"private_key": "0x..."}`).
- mnemonic: BIP39 mnemonic phrase (direct value).
- mnemonic_path: path to file containing the BIP39 mnemonic phrase.
- timeout (nanoseconds): used to cancel retry when fail submissions or retrievals (optional).
- network_id: Network identifier (11155111 for Sepolia, 1 for mainnet).
- api_url: Beacon API Endpoint Link used to retrieve blobs.
- gas_limit: max gas to be used in blob txs (optional).

The address is derived from the mnemonic using the Ethereum BIP44 derivation path `m/44'/60'/0'/0/0`.

## Batch Size Limit

**⚠️ Maximum blob size: ~130,000 bytes (~130KB)**

This is below the default rollapp batch size (500KB). You **must** set `batch_max_size_bytes` in dymint.toml to a value below 130,000 (e.g., `batch_max_size_bytes = "120000"`).
