BUILDDIR?=$(CURDIR)/build
OUTPUT?=$(BUILDDIR)/dymint

BUILD_TAGS?=dymint
CGO_ENABLED ?= 0
VERSION ?= $(shell git describe --tags --always)
LD_FLAGS = -X github.com/dymensionxyz/dymint/version.BuildVersion=$(VERSION)
BUILD_FLAGS = -mod=readonly -ldflags "$(LD_FLAGS)"

# Process Docker environment varible TARGETPLATFORM 
# in order to build binary with correspondent ARCH
# by default will always build for linux/amd64
TARGETPLATFORM ?= 
GOOS ?= linux
GOARCH ?= amd64
GOARM ?=

ifeq (linux/arm,$(findstring linux/arm,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=arm
	GOARM=7
endif

ifeq (linux/arm/v6,$(findstring linux/arm/v6,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=arm
	GOARM=6
endif

ifeq (linux/arm64,$(findstring linux/arm64,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=arm64
	GOARM=7
endif

ifeq (linux/386,$(findstring linux/386,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=386
endif

ifeq (linux/amd64,$(findstring linux/amd64,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=amd64
endif

ifeq (linux/mips,$(findstring linux/mips,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=mips
endif

ifeq (linux/mipsle,$(findstring linux/mipsle,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=mipsle
endif

ifeq (linux/mips64,$(findstring linux/mips64,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=mips64
endif

ifeq (linux/mips64le,$(findstring linux/mips64le,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=mips64le
endif

ifeq (linux/riscv64,$(findstring linux/riscv64,$(TARGETPLATFORM)))
	GOOS=linux
	GOARCH=riscv64
endif

all: check build test install
.PHONY: all

include tests.mk

###############################################################################
###                                Mocks                                    ###
###############################################################################

mock-gen:
	go install github.com/vektra/mockery/v2@latest
	~/go/bin/mockery
.PHONY: mock-gen

###############################################################################
###                                Build Dymint                            ###
###############################################################################

build:
	CGO_ENABLED=$(CGO_ENABLED) go build $(BUILD_FLAGS) -tags '$(BUILD_TAGS)' -o $(OUTPUT) ./cmd/dymint
.PHONY: build

install:
	CGO_ENABLED=$(CGO_ENABLED) go install $(BUILD_FLAGS) -tags $(BUILD_TAGS) ./cmd/dymint
.PHONY: install

###############################################################################
###                                Protobuf                                 ###
###############################################################################

proto-gen:
	@echo "Generating Protobuf files.."
	./proto/gen.sh
.PHONY: proto-gen
