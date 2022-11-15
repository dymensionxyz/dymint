#!/usr/bin/env bash

set -eo pipefail

# Generate the `types` proto files
buf generate --path="./proto/types/dalc" --template="buf.gen.yaml" --config="buf.yaml"
buf generate --path="./proto/types/dymint" --template="buf.gen.yaml" --config="buf.yaml"

# Generate the `test` proto files
buf generate --path="./proto/test" --template="buf.gen.yaml" --config="buf.yaml"