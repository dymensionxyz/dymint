# DA Instructions

## Requirements

- Access to Solana RPC node. Solana DA requires high RPC usage, therefore public RPC is not recommended because rate limitations. Private RPCs, such as <https://www.helius.dev/solana-rpc-nodes> can be used (free plan can be used for testing).
- A Solana account with funds. Faucent for Solana faucet here: <https://faucet.solana.com/>

## Dymint.toml Configuration

- Example:

```shell
da_layer = ['solana']
da_config = ['{"endpoint":"https://devnet.helius-rpc.com","keypath":"/path/to/solana_key.json","program_address":"5cfjxBnFMoqdbZXTMHaoXfQm7obMpYMnkT681sRd95Qo","api_key":"your-api-key","tx_rate_second":5,"req_rate_second":5}']
```

where:

- endpoint: RPC url address for Solana RPC. When using rate limited RPC it is mandatory to set tx_rate_second and req_rate_second according to service limits. In case of deploying local node, it can be directly used node url (e.g. <http://localhost:8899>)
- keypath: path to JSON file containing the private key (format: `{"private_key": "<base58-encoded-keypair>"}`).
- program_address: Address of the program deployed on Solana used to store Rollapp data. The program address to be used by default will be 5cfjxBnFMoqdbZXTMHaoXfQm7obMpYMnkT681sRd95Qo for both mainnet and devnet.
- timeout (nanoseconds): used to cancel retry when fail submissions or retrievals (optional).
- api_key: API key/auth token required by paid RPC nodes like Helius (optional).
- tx_rate_second: rate used to send Solana transactions, required if RPC is rate limited (optional).
- req_rate_second: rate used to send Solana queries, required if RPC is rate limited (optional).
