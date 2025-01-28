#!/usr/bin/env bash

cd "$(dirname "${BASH_SOURCE[0]}")"

TM_VERSION=v0.34.14
TM_PROTO_URL=https://raw.githubusercontent.com/tendermint/tendermint/$TM_VERSION/proto/tendermint

TM_PROTO_FILES=(
  abci/types.proto
  version/types.proto
  types/types.proto
  types/evidence.proto
  types/params.proto
  types/validator.proto
  state/types.proto
  crypto/proof.proto
  crypto/keys.proto
  libs/bits/types.proto
  p2p/types.proto
)

echo Fetching protobuf dependencies from Tendermint $TM_VERSION
for FILE in "${TM_PROTO_FILES[@]}"; do
  echo Fetching "$FILE"
  mkdir -p "tendermint/$(dirname $FILE)"
  curl -sSL "$TM_PROTO_URL/$FILE" > "tendermint/$FILE"
done

COSMOS_VERSION=v0.46.16
COSMOS_PROTO_URL=https://raw.githubusercontent.com/cosmos/cosmos-sdk/$COSMOS_VERSION/proto/cosmos

COSMOS_PROTO_FILES=(
  base/v1beta1/coin.proto
  base/query/v1beta1/pagination.proto
  msg/v1/msg.proto
  bank/v1beta1/bank.proto
)

echo Fetching protobuf dependencies from Cosmos $COSMOS_VERSION
for FILE in "${COSMOS_PROTO_FILES[@]}"; do
  echo Fetching "$FILE"
  mkdir -p "cosmos/$(dirname $FILE)"
  curl -sSL "$COSMOS_PROTO_URL/$FILE" > "cosmos/$FILE"
done

IBC_VERSION=v6.2.1
IBC_PROTO_URL=https://raw.githubusercontent.com/cosmos/ibc-go/refs/tags/$IBC_VERSION/proto

IBC_PROTO_FILES=(
  ibc/applications/transfer/v2/packet.proto
)

echo Fetching protobuf dependencies from IBC $IBC_VERSION
for FILE in "${IBC_PROTO_FILES[@]}"; do
  echo Fetching "$FILE"
  mkdir -p "ibc/$(dirname $FILE)"
  curl -sSL "$IBC_PROTO_URL/$FILE" > "ibc/$FILE"
done
