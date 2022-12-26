#!/usr/bin/make -f

########################################
### Testing

BINDIR ?= $(GOPATH)/bin

### go tests
test:
	@echo "--> Running go test"
	@go test $(PACKAGES) -race
.PHONY: test
