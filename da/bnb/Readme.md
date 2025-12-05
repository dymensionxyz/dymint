# DA Instructions

## Requirements

- Access to a BNB Smart chain RPC node and its url. Public one's can be found here: <https://docs.bnbchain.org/bnb-smart-chain/developers/json_rpc/json-rpc-endpoint/>
- A BNB Smart chain account with funds.

## Dymint.toml Configuration

- Example using environment variable:

```shell
da_layer = ['bnb']
da_config = ['{​"endpoint":"https://data-seed-prebsc-1-s1.bnbchain.org:8545","chain_id":97,"private_key_env":"BNB_PRIVATE_KEY"}']
```

- Example using file (recommended for production):

```shell
da_layer = ['bnb']
da_config = ['{​"endpoint":"https://data-seed-prebsc-1-s1.bnbchain.org:8545","chain_id":97,"private_key_file":"/path/to/private_key.txt"}']
```

- Example with private key directly in config (alternative, not recommended for production):

```shell
da_layer = ['bnb']
da_config = ['{​"endpoint":"https://data-seed-prebsc-1-s1.bnbchain.org:8545","chain_id":97,"private_key_hex":"54a4d5917cf1827a39cebaa9621aed61bdbf82b800aafa91960d16edaf6911a7"}']
```

where:

- private_key_env: (Optional) Environment variable name for private key (e.g., BNB_PRIVATE_KEY). Takes precedence over other methods.
- private_key_file: (Optional) Path to file containing the private key. Takes precedence over `private_key_hex` field.
- private_key_hex: (Optional) Private key directly in config. Used as fallback if neither `private_key_env` nor `private_key_file` is set.
- chain_id = Identifier depending on the network used. 56 for Mainnet and 97 for Testnet.
- endpoint: BNB smart chain rpc node url address. Public rpc's are available here: <https://docs.bnbchain.org/bnb-smart-chain/developers/json_rpc/json-rpc-endpoint/#rpc-endpoints-for-bnb-smart-chain>
