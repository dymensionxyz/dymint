# dymint

ABCI-client implementation for dYmenion's autonomous RollApp forked from [celestiaorg/optimint](https://github.com/celestiaorg/optimint).

To learn more about dYmension's autonomous RollApps and dymint read the [docs](https://docs.dymension.xyz/learn/rollapps)

## Building From Source

Requires Go version >= 1.18. If you need to install Go on your system, head to the [Go download and install page](https://go.dev/doc/install).

To build:

```sh
git clone https://github.com/dymensionxyz/dymint.git
cd dymint
go build -v ./...
```

To test:

```sh
go test ./...
```

To regenerate protobuf types:

```sh
./proto/gen.sh
```
