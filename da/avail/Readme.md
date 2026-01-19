# DA Instructions

## Requirements

- Access to an Avail RPC node and its url.
- An Avail account with funds.
- Registration of an App id for the specific RollApp <https://docs.availproject.org/docs/learn-about-avail/app-ids>

## Dymint.toml Configuration

- Example using environment variable:

```shell
da_layer = ['avail']
da_config = ['{​"seed_env":"AVAIL_SEED","endpoint":"https://turing-rpc.avail.so/rpc","app_id":1}']
```

- Example using file (recommended for production):

```shell
da_layer = ['avail']
da_config = ['{​"seed_file":"/path/to/seed.txt","endpoint":"https://turing-rpc.avail.so/rpc","app_id":1}']
```

- Example with seed directly in config (alternative, not recommended for production):

```shell
da_layer = ['avail']
da_config = ['{​"seed":"talent improve history affair neck gadget flock blossom brick post behind sheriff pole exchange income solve stage hundred soldier balcony cloud happy clean noble","endpoint":"https://turing-rpc.avail.so/rpc","app_id":1}']
```

where:

- seed_env: (Optional) Environment variable name for seed (e.g., AVAIL_SEED). Takes precedence over other methods.
- seed_file: (Optional) Path to file containing the seed. Takes precedence over `seed` field.
- seed: (Optional) Seed phrase directly in config. Used as fallback if neither `seed_env` nor `seed_file` is set.
- endpoint = avail rpc node url address. Public rpc's are available here: <https://avail-rpc.publicnode.com/>
- app_id: identifier of the RollApp in Avail, that needs to be registered beforehand. More info in here: <https://docs.availproject.org/docs/learn-about-avail/app-ids>
