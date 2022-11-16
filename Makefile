Version := $(shell cat VERSION)
GitCommit := $(shell git rev-parse HEAD)
CompileTime := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
DIR := $(shell pwd)
LDFLAGS := -s -w -X main.Version=$(Version) -X main.GitCommit=$(GitCommit) -X main.CompileTime=$(CompileTime)

build:
	go build -race -ldflags "$(LDFLAGS)" -o build/debug/heimdall main.go

release:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/heimdall-linux main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/heimdall-darwin main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o build/release/heimdall-darwin-m1 main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/heimdall.exe main.go

.PHONY: build release
