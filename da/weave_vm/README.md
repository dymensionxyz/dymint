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