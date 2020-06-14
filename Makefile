SHELL := /bin/bash
version := $(shell git rev-list --count HEAD)
commit := $(shell git describe --always --long --dirty)
built_at := $(shell date +%FT%T%z)
built_by := ${USER}

flags := -gcflags="all=-N -l -c 2"
ldflags := -X main.version=v${version} -X main.commit=${commit}
ldflags += -X main.builtAt=${built_at} -X main.builtBy=${built_by}

dist := ./dist
env := GO111MODULE=on
DIR := ${CURDIR}

ddos:
	$(env) go build -o $(dist)/ddos -ldflags="$(ldflags)" cmd/seednodeddos/main.go

eclipse:
	$(env) go build -o $(dist)/eclipse -ldflags="$(ldflags)" cmd/eclipse/main.go

spam:
	$(env) go build -o $(dist)/spam -ldflags="$(ldflags)" cmd/spam/main.go

.PHONY:clean

clean:
	@rm -rf ./dist
