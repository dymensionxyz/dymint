# DA Instructions

## Requirements

- Access to a Sui RPC node. Public endpoints: <https://fullnode.devnet.sui.io:443> (devnet), <https://fullnode.testnet.sui.io:443> (testnet), <https://fullnode.mainnet.sui.io:443> (mainnet).
- A Sui account with funds. Faucet for devnet/testnet: <https://faucet.sui.io/>.
- A deployed noop contract address.

## Deploying the Noop Contract

The noop contract must be deployed before using Sui DA. Sui devnet/testnet are reset periodically, so you may need to redeploy.

1. Install Sui CLI: <https://docs.sui.io/guides/developer/getting-started/sui-install>

2. Configure Sui CLI for the target network:

   ```shell
   sui client switch --env devnet
   ```

3. Import your wallet (using the same mnemonic as DA config):

   ```shell
   sui keytool import "<your-mnemonic>" ed25519
   ```

4. Get funds from faucet: <https://faucet.sui.io/>

5. Deploy the contract from the `noop` directory:

   ```shell
   cd da/sui/noop
   sui client publish --gas-budget 100000000
   ```

6. From the output, find the "Published Objects" section and copy the package ID. Use this as `noop_contract_address` in your config.

## Dymint.toml Configuration

- Example using mnemonic directly:

```shell
da_layer = ['sui']
da_config = ['{"endpoint":"https://fullnode.devnet.sui.io:443","noop_contract_address":"0x3dbdaa3db8d587deb38be3d4825ff434f1723054a6f43e04c0623f2c21a3f8a2","gas_budget":"10000000","mnemonic":"your twelve word mnemonic phrase here"}']
```

- Example using mnemonic from file:

```shell
da_layer = ['sui']
da_config = ['{"endpoint":"https://fullnode.devnet.sui.io:443","noop_contract_address":"0x3dbdaa3db8d587deb38be3d4825ff434f1723054a6f43e04c0623f2c21a3f8a2","gas_budget":"10000000","mnemonic_path":"/path/to/mnemonic.txt"}']
```

where:

- endpoint: Sui RPC node url address.
- noop_contract_address: Address of the noop contract deployed on Sui used to store data.
- gas_budget: Gas budget for transactions (default: 10000000 = 0.01 SUI).
- mnemonic: BIP39 mnemonic phrase (direct value).
- mnemonic_path: path to file containing the BIP39 mnemonic phrase.

The address is derived from the mnemonic using ed25519 key derivation.

## Batch Size Limit

Maximum blob size: ~96,000 bytes (~96KB)

This is below the default rollapp batch size (500KB). You **must** set `batch_submit_bytes` in dymint.toml to a value below 96,000 (e.g., `batch_submit_bytes = 90000`).
