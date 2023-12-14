#!/bin/bash
rm -rf /Users/sergi/.dymint/
rm -rf /Users/sergi/.dymint2/
./build/dymint init
./build/dymint init --home=/Users/sergi/.dymint2
rm /Users/sergi/.dymint2/config/genesis.json
rm /Users/sergi/.dymint2/config/priv_validator_key.json
cp /Users/sergi/.dymint/config/genesis.json /Users/sergi/.dymint2/config/
cp /Users/sergi/.dymint/config/priv_validator_key.json /Users/sergi/.dymint2/config/
