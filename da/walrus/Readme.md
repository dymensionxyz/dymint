# DA Instructions

## Requirements

- Access to Walrus publisher and aggregator endpoints (public testnet endpoints available).
- No account/funds needed - Walrus uses a public publisher.

## Dymint.toml Configuration

- Example:

```shell
da_layer = ['walrus']
da_config = ['{"publisher_url":"https://publisher.walrus-testnet.walrus.space","aggregator_url":"https://aggregator.walrus-testnet.walrus.space","blob_owner_addr":"0xcc7f20e6ca6d5b9076068bf9b40421218fdf2cfa6316f48c428c8b6716db9c05","store_duration_epochs":53}']
```

where:

- publisher_url: Walrus publisher endpoint URL.
- aggregator_url: Walrus aggregator endpoint URL.
- blob_owner_addr: Address that owns the blobs.
- store_duration_epochs: Number of epochs to store the data (max 53 for testnet).
- timeout (nanoseconds): used to cancel retry when fail submissions or retrievals (optional, default: 5 minutes).

Note: Walrus uses a public publisher, so no private key or mnemonic is required.
