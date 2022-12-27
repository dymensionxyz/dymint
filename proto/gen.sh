#!/usr/bin/env bash

# see: https://stackoverflow.com/a/246128
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"/..

# Target will hold the target dirs to generate proto files for
TARGETS=()

# Build target dirs from the proto dir
for d in ./proto/* ; do
     if [ -d "$d" ]; then
        # Add $d to the list of targets
        TARGET="${d}/pb"
        # Remove the parent directory
        TARGET=${TARGET#*/*/}
        # Add to the list of targets
        TARGETS="$TARGETS ./$TARGET"
    fi
done


# Iterate over the target dirs and delete the current contents
for TARGET_DIR in $TARGETS; do
    echo "Deleting contents of $TARGET_DIR"
    rm -rf $TARGET_DIR/*
done

# Generate proto files to ./proto/pb
cd $SCRIPT_DIR
mkdir -p ./proto/pb
rm -rf $TARGET_DIR/*
docker run -v $PWD:/workspace --workdir /workspace tendermintdev/docker-build-proto sh ./proto/protoc.sh

# Copy the generated files to the target directories 
for TARGET_DIR in $TARGETS; do
    echo "Copying generated proto files to $TARGET_DIR"
    TARGET_DIR_BASE_DIR=$(echo "$TARGET_DIR" | cut -d "/" -f2)
    cp -r ./proto/pb/$TARGET_DIR_BASE_DIR/* $TARGET_DIR
done

echo "Deleting proto pb dir"
rm -rf ./proto/pb
