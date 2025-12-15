# DA Instructions

## Requirements

- Access to a BNB Smart chain RPC node and its url. Public one's can be found here: <https://docs.bnbchain.org/bnb-smart-chain/developers/json_rpc/json-rpc-endpoint/>
- A BNB Smart chain account with funds.

## Dymint.toml Configuration

- Example using key file:

```shell
da_layer = ['bnb']
da_config = ['{"endpoint":"https://data-seed-prebsc-1-s1.bnbchain.org:8545","network_id":97,"keypath":"/path/to/bnb_key.json"}']
```

- Example using mnemonic directly:

```shell
da_layer = ['bnb']
da_config = ['{"endpoint":"https://data-seed-prebsc-1-s1.bnbchain.org:8545","network_id":97,"mnemonic":"your twelve word mnemonic phrase here"}']
```

- Example using mnemonic from file:

```shell
da_layer = ['bnb']
da_config = ['{"endpoint":"https://data-seed-prebsc-1-s1.bnbchain.org:8545","network_id":97,"mnemonic_path":"/path/to/mnemonic.txt"}']
```

where:

- keypath: path to JSON file containing the private key (format: `{"private_key": "0x..."}`).
- mnemonic: BIP39 mnemonic phrase (direct value).
- mnemonic_path: path to file containing the BIP39 mnemonic phrase.
- network_id: Identifier depending on the network used. 56 for Mainnet and 97 for Testnet.
- endpoint: BNB smart chain rpc node url address. Public rpc's are available here: <https://docs.bnbchain.org/bnb-smart-chain/developers/json_rpc/json-rpc-endpoint/#rpc-endpoints-for-bnb-smart-chain>

The address is derived from the mnemonic using the Ethereum BIP44 derivation path `m/44'/60'/0'/0/0`.
