# DA Instructions

## Requirements

- Access to a Kaspa node via grpc. Instructions on how to run Kaspa node <https://github.com/kaspanet/rusty-kaspa/>
- Access to an indexer (<https://github.com/supertypo/simply-kaspa-indexer>) archive node, supporting Kaspa API <https://api.kaspa.org/docs> (<https://github.com/kaspa-ng/kaspa-rest-server>).
- A Kaspa account with funds.

## Dymint.toml Configuration

- Example:

```shell
da_layer = ['kaspa']
da_config = ['{"api_url":"https://api-tn10.kaspa.org","grpc_address":"localhost:16210","network":"kaspa-testnet-10","address":"kaspatest:qzwyrgapjnhtjqkxdrmp7fpm3yddw296v2ajv9nmgmw5k3z0r38guevxyk7j0","mnemonic_env":"KASPA_MNEMONIC"}']
```

where:

- api_url = URL for archive node supporting Kaspa API. There are public endpoints available at <https://api-tn10.kaspa.org> for Testnet-10 or <https://api.kaspa.org> for Mainnet.
- grpc_address = IP and port of a Kaspa Node to access via grpc.
- network: kaspa-testnet-10 or kaspa-mainnet.
- address: Kaspa funded address.
- mnemonic_env: Env variable used to set account mnemonic, e.g. KASPA_MNEMONIC.
