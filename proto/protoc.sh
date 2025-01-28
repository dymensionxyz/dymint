#!/usr/bin/env bash

set -eo pipefail

# Generate the `types` proto files
buf generate --path="./proto/types/dalc" --template="buf.gen.yaml" --config="buf.yaml"
buf generate --path="./proto/types/dymint" --template="buf.gen.yaml" --config="buf.yaml"
buf generate --path="./proto/types/interchain_da" --template="buf.gen.yaml" --config="buf.yaml"
buf generate --path="./proto/types/dymensionxyz" --template="buf.gen.yaml" --config="buf.yaml"

# Generate the `test` proto files
buf generate --path="./proto/test" --template="buf.gen.yaml" --config="buf.yaml"

# Generate the 'p2p' proto files
buf generate --path="./proto/p2p" --template="buf.gen.yaml" --config="buf.yaml"