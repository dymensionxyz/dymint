# DA Instructions

## Requirements

- Access to Solana RPC node. Solana DA requires high RPC usage, therefore public RPC is not recommended because rate limitations. Private RPCs, such as <https://www.helius.dev/solana-rpc-nodes> can be used (free plan can be used for testing).
- A Solana account with funds. Faucent for Solana faucet here: <https://faucet.solana.com/>

## Dymint.toml Configuration

- Example:

```shell
da_layer = ['solana']
da_config = ['{"endpoint":"https://devnet.helius-rpc.com/?api-key=ffd685bc-4ce6-4e58-95ef-9924cebba707","keypath_env":"SOLANA_KEYPATH","program_address":"5cfjxBnFMoqdbZXTMHaoXfQm7obMpYMnkT681sRd95Qo","tx_rate_second":5,"req_rate_second":5}']
```

where:

- endpoint = RPC url address for Solana RPC. This is an example of Helius RPC service with api-key included. When using rate limited RPC it is mandatory to set tx_rate_second and req_rate_second according to service limits. In case of deploying local node, it can be directly used node url (e.g. <http://localhost:8899>)
- keypath_env = env variable name used to set the path where the solana private key is stored.
- program_address: sequencer Solana address with funds.
- timeout (nanoseconds): used to cancel retry when fail submissions or retrievals (optional).
- apikey_env: env variable used to set the auth token required by RPC nodes, if necessary (optional).
- tx_rate_second: rate used to send Solana transactions, required if RPC is rate limited (optional).
- req_rate_second: rate used to send Solana queries, required if RPC is rate limited (optional).
