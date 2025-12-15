# DA Instructions

## Requirements

- Access to a Kaspa node via grpc. Instructions on how to run Kaspa node <https://github.com/kaspanet/rusty-kaspa/>
- Access to an indexer (<https://github.com/supertypo/simply-kaspa-indexer>) archive node, supporting Kaspa API <https://api.kaspa.org/docs> (<https://github.com/kaspa-ng/kaspa-rest-server>).
- A Kaspa account with funds (funded at the default derivation path `m/44'/111111'/0'/0/0`).

## Dymint.toml Configuration

- Example:

```shell
da_layer = ['kaspa']
da_config = ['{"api_url":"https://api-tn10.kaspa.org","grpc_address":"localhost:16210","network":"kaspa-testnet-10","mnemonic":"your twelve word mnemonic phrase here"}']
```

where:

- api_url = URL for archive node supporting Kaspa API. There are public endpoints available at <https://api-tn10.kaspa.org> for Testnet-10 or <https://api.kaspa.org> for Mainnet.
- grpc_address = IP and port of a Kaspa Node to access via grpc.
- network: kaspa-testnet-10 or kaspa-mainnet.
- mnemonic: BIP39 mnemonic phrase (can also use mnemonic_path to read from file).

The address is automatically derived from the mnemonic using the standard Kaspa BIP44 derivation path `m/44'/111111'/0'/0/0`.
