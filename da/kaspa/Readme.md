# DA Instructions

## Requirements

- Access to a Kaspa node via grpc. Instructions on how to run Kaspa node <https://github.com/kaspanet/rusty-kaspa/>
- Access to an indexer (<https://github.com/supertypo/simply-kaspa-indexer>) archive node, supporting Kaspa API <https://api.kaspa.org/docs> (<https://github.com/kaspa-ng/kaspa-rest-server>).
- A Kaspa account with funds.

## Dymint.toml Configuration

- Example using environment variable:

```shell
da_layer = ['kaspa']
da_config = ['{"api_url":"https://api-tn10.kaspa.org","grpc_address":"localhost:16210","network":"kaspa-testnet-10","address":"kaspatest:qzwyrgapjnhtjqkxdrmp7fpm3yddw296v2ajv9nmgmw5k3z0r38guevxyk7j0","mnemonic_env":"KASPA_MNEMONIC"}']
```

- Example with mnemonic directly in config (alternative to environment variable):

```shell
da_layer = ['kaspa']
da_config = ['{"api_url":"https://api-tn10.kaspa.org","grpc_address":"localhost:16210","network":"kaspa-testnet-10","address":"kaspatest:qzwyrgapjnhtjqkxdrmp7fpm3yddw296v2ajv9nmgmw5k3z0r38guevxyk7j0","mnemonic":"your mnemonic words here"}']
```

where:

- api_url = URL for archive node supporting Kaspa API. There are public endpoints available at <https://api-tn10.kaspa.org> for Testnet-10 or <https://api.kaspa.org> for Mainnet.
- grpc_address = IP and port of a Kaspa Node to access via grpc.
- network: kaspa-testnet-10 or kaspa-mainnet.
- address: Kaspa funded address.
- mnemonic_env: (Optional) Environment variable name for mnemonic (e.g., KASPA_MNEMONIC). Takes precedence over `mnemonic` field.
- mnemonic: (Optional) Mnemonic phrase directly in config. Used as fallback if `mnemonic_env` is not set.
