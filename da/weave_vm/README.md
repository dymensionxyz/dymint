## WEAVE VM DA CLIENT 

### Key Details

* WeaveVM provides a gateway for Arweave's permanent with its own (WeaveVM) high data throughput of the permanently stored data into .
* Current maximum encoded blob size is 8 MB (8_388_608 bytes).
* WeaveVM currently operating in public testnet (Alphanet) - not recommended to use it in production environment.

### How it works

When transaction settles on weaveVM chain it has not only weaveVM guarantess but it also will be eventually included in Arweave.
It allows decentralized and secure way to permanently store RollApps data.

### Links

https://docs.wvm.dev/
https://explorer.wvm.dev/
https://www.wvm.dev/faucet

### How To
You should chose "weaveVM" as a DA layer of your RollApp.

To successfully store data on weaveVM as a first step you need to obtain test tWVM tokens through our faucet.
If you are going to use private key of your weaveVM address to sign transactions your config will look like this

```toml
da_config = '{"endpoint":"https://testnet-rpc.wvm.dev","chain_id":9496,"timeout":60000000000,"private_key_hex":"your_hex_string_wvm_priv_key_without_0x_at_the_beginning"}'
```

Using with web3signer as an external signer:
```toml
da_config = '{"endpoint":"https://testnet-rpc.wvm.dev","chain_id":9496,"timeout":"60000000000","web3_signer_endpoint":"http://localhost:9000"}'
```

to enable tls you should add next fields to the json:

```
web3_signer_tls_cert_file
web3_signer_tls_key_file
web3_signer_tls_ca_cert_file
```

## Example of rollap-evm configuration

Please refer to the most recent configuration in rollap-evm repo.
But this can be used as an example.

```sh
# Set environment variables
export DA_CLIENT="weavevm"  # This is the key change

export ROLLAPP_CHAIN_ID="rollappevm_1234-1"
export KEY_NAME_ROLLAPP="rol-user"
export BASE_DENOM="arax"
export MONIKER="$ROLLAPP_CHAIN_ID-sequencer"
export ROLLAPP_HOME_DIR="$HOME/.rollapp_evm"
export SETTLEMENT_LAYER="mock"

# Initialize and start
make install BECH32_PREFIX=$BECH32_PREFIX
export EXECUTABLE="rollapp-evm"
$EXECUTABLE config keyring-backend test

sh scripts/init.sh

# Verify dymint.toml configuration
cat $ROLLAPP_HOME_DIR/config/dymint.toml | grep -A 5 "da_config"

dasel put -f "${ROLLAPP_HOME_DIR}"/config/dymint.toml "max_idle_time" -v "2s"
dasel put -f "${ROLLAPP_HOME_DIR}"/config/dymint.toml "max_proof_time" -v "1s"
dasel put -f "${ROLLAPP_HOME_DIR}"/config/dymint.toml "batch_submit_time" -v "30s"
dasel put -f "${ROLLAPP_HOME_DIR}"/config/dymint.toml "p2p_listen_address" -v "/ip4/0.0.0.0/tcp/36656"
dasel put -f "${ROLLAPP_HOME_DIR}"/config/dymint.toml "settlement_layer" -v "mock"
dasel put -f "${ROLLAPP_HOME_DIR}"/config/dymint.toml "node_address" -v "http://localhost:36657"
dasel put -f "${ROLLAPP_HOME_DIR}"/config/dymint.toml "settlement_node_address" -v "http://127.0.0.1:36657"


# Start the rollapp

$EXECUTABLE start --log_level=debug \
  --rpc.laddr="tcp://127.0.0.1:36657" \
  --p2p.laddr="tcp://0.0.0.0:36656" \
  --proxy_app="tcp://127.0.0.1:36658"
```