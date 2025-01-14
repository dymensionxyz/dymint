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
export WVM_PRIV_KEY="your_hex_string_wvm_priv_key_without_0x_at_the_beginning"

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

in rollap-evm log you will eventually see something like this:
```log
INFO[0000] weaveVM: successfully sent transaction[tx hash 0x8a7a7f965019cf9d2cc5a3d01ee99d56ccd38977edc636cc0bbd0af5d2383d2a]  module=weavevm
INFO[0000] wvm tx hash[hash 0x8a7a7f965019cf9d2cc5a3d01ee99d56ccd38977edc636cc0bbd0af5d2383d2a]  module=weavevm
DEBU[0000] waiting for receipt[txHash 0x8a7a7f965019cf9d2cc5a3d01ee99d56ccd38977edc636cc0bbd0af5d2383d2a attempt 0 error get receipt failed: failed to get transaction receipt: not found]  module=weavevm
INFO[0002] Block created.[height 35 num_tx 0 size 786]   module=block_manager
DEBU[0002] Applying block[height 35 source produced]     module=block_manager
DEBU[0002] block-sync advertise block[error failed to find any peer in table]  module=p2p
INFO[0002] MINUTE EPOCH 6[]                              module=x/epochs
INFO[0002] Epoch Start Time: 2025-01-13 09:21:03.239539 +0000 UTC[]  module=x/epochs
INFO[0002] commit synced[commit 436F6D6D697449447B5B3130342038203131302032303620352031323920393020343520313633203933203235322031352031343320333920313538203131342035382035352031352038322038203939203132392032333520313731203230382031392032343320313932203139203233352036355D3A32337D]
DEBU[0002] snapshot is skipped[height 35]
INFO[0002] Gossipping block[height 35]                   module=block_manager
DEBU[0002] Gossiping block.[len 792]                     module=p2p
DEBU[0002] indexed block[height 35]                      module=txindex
DEBU[0002] indexed block txs[height 35 num_txs 0]        module=txindex
INFO[0002] Produced empty block.[]                       module=block_manager
DEBU[0002] Added bytes produced to bytes pending submission counter.[bytes added 786 pending 15719]  module=block_manager
INFO[0003] data available in weavevm[wvm_tx 0x8a7a7f965019cf9d2cc5a3d01ee99d56ccd38977edc636cc0bbd0af5d2383d2a wvm_block 0xe897eab56aee50b97a0f2bd1ff47af3c834e96ca18528bb869c4eafc0df583be wvm_block_number 5651207]  module=weavevm
DEBU[0003] Submitted blob to DA successfully.[]          module=weavevm
```