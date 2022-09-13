# dymint

ABCI-client implementation for Optimistic Rollups on top of dymension settlement hub.

## Building From Source

Requires Go version >= 1.17. If you need to install Go on your system, head to the [Go download and install page](https://go.dev/doc/install).

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
