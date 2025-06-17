# DA Instructions

## Requirements

- Run a Celestia light node, following these instructions <https://docs.celestia.org/how-to-guides/light-node>
- Testnet RollApps require using Mocha network, and Mainnet RollApps Mainnet Beta.
- Light nodes can be fast synced by specifying in <celestia_folder/config.toml> which block it should start syncing from. For that a trusted block hash (obtained from <https://celenium.io/blocks>) needs to be added in [Header] TrustedHash field, and the block id in [DASer] SampleFrom field.
- Light client account needs to be funded. To know the address run:

```shell
celestia state account-address --node.store <celestia_folder>
```

## Dymint.toml Configuration

```shell
da_layer = ['celestia']
da_config = ['{"base_url": "http://localhost:26658", "timeout": 60000000000, "gas_prices":1.0, "namespace_id": "d6c42bf7dc5b9559623f", "auth_token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJBbGxvdyI6WyJwdWJsaWMiLCJyZWFkIiwid3JpdGUiLCJhZG1pbiJdLCJOb25jZSI6IkI1bm9WVVBZbG9hcWY2MlVBTEp1aWxGaWQxVmNSK0JKYlk1WDVPTi9MZWs9IiwiRXhwaXJlc0F0IjoiMDAwMS0wMS0wMVQwMDowMDowMFoifQ.e2siNJTo9YoO1oqNausuwUL-8SJrtN4GhY4_iLyVw8E"}']
```

where:

- base_url = url pointing to the light node, use localhost when running in same machine, otherwise specify the right IP address.
- timeout (nanoseconds)= used to cancel retry when fail submissions or retrievals.
- gas_prices: can be adjusted based on gas prices <https://celenium.io/gas>
- namespace_id: Namespace used to identify RollApps within Celestia <https://celestiaorg.github.io/celestia-app/namespace.html>
- auth_token: auth token issues by the light client. You can get it running the following command:

```shell
celestia light auth admin --p2p.network <network>
```
