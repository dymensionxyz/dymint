# DA Instructions

## Requirements

- Access to a Sui RPC node (e.g., <https://fullnode.devnet.sui.io:443> for devnet, <https://fullnode.mainnet.sui.io:443> for mainnet)
- A Sui account with funds. Faucet for devnet can be accessed via SDK: `sui.RequestSuiFromFaucet("https://faucet.devnet.sui.io", address, map[string]string{})`
- A deployed Noop contract address

## Dymint.toml Configuration

- Example using environment variable:

```shell
da_layer = ['sui']
da_config = ['{​"rpc_url":"https://fullnode.devnet.sui.io:443","noop_contract_address":"0x45d86eb334f15b3a5145c0b7012dae3bf16de58ab4777ae31d184e9baf91c420","gas_budget":"10000000","mnemonic_env":"SUI_MNEMONIC"}']
```

- Example using file (recommended for production):

```shell
da_layer = ['sui']
da_config = ['{​"rpc_url":"https://fullnode.devnet.sui.io:443","noop_contract_address":"0x45d86eb334f15b3a5145c0b7012dae3bf16de58ab4777ae31d184e9baf91c420","gas_budget":"10000000","mnemonic_file":"/path/to/mnemonic.txt"}']
```

- Example with mnemonic directly in config (alternative, not recommended for production):

```shell
da_layer = ['sui']
da_config = ['{​"rpc_url":"https://fullnode.devnet.sui.io:443","noop_contract_address":"0x45d86eb334f15b3a5145c0b7012dae3bf16de58ab4777ae31d184e9baf91c420","gas_budget":"10000000","mnemonic":"your mnemonic words here"}']
```

where:

- rpc_url: Sui RPC endpoint URL
- noop_contract_address: Address of the deployed Noop smart contract
- gas_budget: Gas budget for transactions (default: "10000000" = 0.01 SUI)
- mnemonic_env: (Optional) Environment variable name for mnemonic (e.g., SUI_MNEMONIC). Takes precedence over other methods.
- mnemonic_file: (Optional) Path to file containing the mnemonic. Takes precedence over `mnemonic` field.
- mnemonic: (Optional) Mnemonic phrase directly in config. Used as fallback if neither `mnemonic_env` nor `mnemonic_file` is set.
- timeout: (Optional) Timeout for RPC requests
- retry_attempts: (Optional) Number of retry attempts for failed operations (default: 5)
- retry_delay: (Optional) Delay between retry attempts (default: 3s)
