# DA Instructions

## Requirements
- Access to a BNB Smart chain RPC node and its url. Public one's can be found here: https://docs.bnbchain.org/bnb-smart-chain/developers/json_rpc/json-rpc-endpoint/
- A BNB Smart chain account with funds.

## Dymint.toml Configuration

* Example:
```shell 
da_layer = ['bnb']
da_config = ['{"endpoint":"https://data-seed-prebsc-1-s1.bnbchain.org:8545","chain_id":97,"key":"54a4d5917cf1827a39cebaa9621aed61bdbf82b800aafa91960d16edaf6911a7"}']
```

where: 

* key = private key corresponding to the funded address.
* chain_id = Identifier depending on the network used. 56 for Mainnet and 97 for Testnet.
* endpoint: BNB smart chain rpc node url address. Public rpc's are available here: https://docs.bnbchain.org/bnb-smart-chain/developers/json_rpc/json-rpc-endpoint/#rpc-endpoints-for-bnb-smart-chain

