
GO_BUILD_ENV :=
GO_BUILD_FLAGS := -tags no_cgo,osusergo,netgo
MODULE_BINARY := triangle_finder

ifeq ($(VIAM_TARGET_OS), windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	MODULE_BINARY = triangle_finder.exe
endif

ifeq ($(VIAM_TARGET_OS),linux)
    GO_BUILD_FLAGS += -ldflags="-extldflags=-static -s -w"
endif

$(MODULE_BINARY): Makefile
	$(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $(MODULE_BINARY) cmd/module/main.go

module.tar.gz: meta.json $(MODULE_BINARY)
	tar czf $@ meta.json $(MODULE_BINARY) templates
	git checkout meta.json

ifneq ($(VIAM_TARGET_OS), windows)
module.tar.gz: strip-module
endif

strip-module: 
	strip $(MODULE_BINARY)

all: module test

update:
	go get go.viam.com/rdk@latest
	go mod tidy

test:
	go test ./...
