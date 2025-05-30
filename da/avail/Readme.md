# DA Instructions

## Requirements
- Access to an Avail RPC node and its url.
- An Avail account with funds.
- Registration of an App id for the specific RollApp https://docs.availproject.org/docs/learn-about-avail/app-ids

## Dymint.toml Configuration

* Example:
```shell 
da_layer = ['avail']
da_config = ['{"{"seed": "talent improve history affair neck gadget flock blossom brick post behind sheriff pole exchange income solve stage hundred soldier balcony cloud happy clean noble", "api_url": "https://turing-rpc.avail.so/rpc", "app_id": 1}']
```

where: 

* seed = seed phrase used to generate the private key of the avail account.
* api_url = avail rpc node url address. Public rpc's are available here: https://avail-rpc.publicnode.com/
* app_id: identifier of the RollApp in Avail, that needs to be registered beforehand. More info in here: https://docs.availproject.org/docs/learn-about-avail/app-ids

