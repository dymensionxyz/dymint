# DA Instructions

## Requirements

- Access to an Avail RPC node and its url.
- An Avail account with funds.
- Registration of an App id for the specific RollApp <https://docs.availproject.org/docs/learn-about-avail/app-ids>

## Dymint.toml Configuration

- Example using mnemonic directly:

```shell
da_layer = ['avail']
da_config = ['{"mnemonic":"talent improve history affair neck gadget flock blossom brick post behind sheriff pole exchange income solve stage hundred soldier balcony cloud happy clean noble","endpoint":"https://turing-rpc.avail.so/rpc","app_id":1}']
```

- Example using mnemonic from file:

```shell
da_layer = ['avail']
da_config = ['{"mnemonic_path":"/path/to/mnemonic.txt","endpoint":"https://turing-rpc.avail.so/rpc","app_id":1}']
```

- Example using key file:

```shell
da_layer = ['avail']
da_config = ['{"keypath":"/path/to/avail_key.json","endpoint":"https://turing-rpc.avail.so/rpc","app_id":1}']
```

where:

- mnemonic: BIP39 mnemonic phrase (direct value).
- mnemonic_path: path to file containing the BIP39 mnemonic phrase.
- keypath: path to JSON file containing the private key (format: `{"private_key": "0x..."}`).
- endpoint: Avail RPC node url address. Public rpc's are available here: <https://avail-rpc.publicnode.com/>
- app_id: identifier of the RollApp in Avail, that needs to be registered beforehand. More info in here: <https://docs.availproject.org/docs/learn-about-avail/app-ids>
