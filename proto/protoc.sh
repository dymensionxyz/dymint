#!/usr/bin/env bash

set -eo pipefail

buf generate --path="./proto/dalc" --template="buf.gen.yaml" --config="buf.yaml"
buf generate --path="./proto/dymint" --template="buf.gen.yaml" --config="buf.yaml"
