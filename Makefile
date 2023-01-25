#! /usr/bin/make -f

API=payment

# Go related variables.
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/bin
GOPKG := $(.)
LDFLAGS=-ldflags "-X=main.Version=$(VERSION) -X=main.Build=$(BUILD)"
# A valid GOPATH is required to use the `go get` command.
# If $GOPATH is not specified, $HOME/go will be used by default
GOPATH := $(if $(GOPATH),$(GOPATH),~/go)

build/api:
	@echo "  >  Building api ..."
	go build -o $(GOBIN)/$(API)/api $(GOBASE)/cmd/api

build/worker:
	@echo "  >  Building worker ..."
	go build -o $(GOBIN)/$(API)/worker $(GOBASE)/cmd/worker