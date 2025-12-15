# DA Instructions

## Requirements

- Access to a Kaspa node via grpc. Instructions on how to run Kaspa node <https://github.com/kaspanet/rusty-kaspa/>
- Access to an indexer (<https://github.com/supertypo/simply-kaspa-indexer>) archive node, supporting Kaspa API <https://api.kaspa.org/docs> (<https://github.com/kaspa-ng/kaspa-rest-server>).
- A Kaspa account with funds (funded at the default derivation path `m/44'/111111'/0'/0/0`).

## Dymint.toml Configuration

- Example using mnemonic directly:

```shell
da_layer = ['kaspa']
da_config = ['{"api_url":"https://api-tn10.kaspa.org","endpoint":"localhost:16210","network_id":"kaspa-testnet-10","mnemonic":"your twelve word mnemonic phrase here"}']
```

- Example using mnemonic from file:

```shell
da_layer = ['kaspa']
da_config = ['{"api_url":"https://api-tn10.kaspa.org","endpoint":"localhost:16210","network_id":"kaspa-testnet-10","mnemonic_path":"/path/to/mnemonic.txt"}']
```

where:

- api_url: URL for archive node supporting Kaspa API. There are public endpoints available at <https://api-tn10.kaspa.org> for Testnet-10 or <https://api.kaspa.org> for Mainnet.
- endpoint: IP and port of a Kaspa Node to access via grpc.
- network_id: kaspa-testnet-10 or kaspa-mainnet.
- mnemonic: BIP39 mnemonic phrase (direct value).
- mnemonic_path: path to file containing the BIP39 mnemonic phrase.

The address is automatically derived from the mnemonic using the standard Kaspa BIP44 derivation path `m/44'/111111'/0'/0/0`.
