# DA Instructions

## Requirements

- Access to Solana RPC node. Solana DA requires high RPC usage, therefore public RPC is not recommended because rate limitations. Private RPCs, such as <https://www.helius.dev/solana-rpc-nodes> can be used (free plan can be used for testing).
- A Solana account with funds. Faucent for Solana faucet here: <https://faucet.solana.com/>

## Dymint.toml Configuration

- Example:

```shell
da_layer = ['solana']
da_config = ['{"endpoint":"http://barcelona:8899","keypath_env":"SOLANA_KEYPATH","program_address":"5cfjxBnFMoqdbZXTMHaoXfQm7obMpYMnkT681sRd95Qo"}']
```

where:

- endpoint = RPC url address for Solana RPC.
- keypath_env = env variable name used to set the path where the solana private key is stored.
- program_address: sequencer Solana address with funds.
- timeout (nanoseconds): used to cancel retry when fail submissions or retrievals (optional).
- apikey_env: env variable used to set the auth token required by RPC nodes, if necessary (optional).
- tx_rate: rate used to send Solana transactions, required if RPC is rate limited (optional).
- req_rate: rate used to send Solana queries, required if RPC is rate limited (optional).
