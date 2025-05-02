GO_BUILD_ENV :=
GO_BUILD_FLAGS :=
MODULE_BINARY := bin/viam-triangle-finder

ifeq ($(VIAM_TARGET_OS), windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	GO_BUILD_FLAGS := -tags no_cgo	
	MODULE_BINARY = bin/viam-hough-transform.exe
endif

ifeq ($(VIAM_TARGET_OS), linux)
	GO_BUILD_ENV += CGO_LDFLAGS='-ltbb'
	GO_BUILD_FLAGS := -tags opencvstatic
endif


$(MODULE_BINARY):
	$(GO_BUILD_ENV) go build $(GO_BUILD_FLAGS) -o $(MODULE_BINARY) cmd/module/main.go

module.tar.gz: meta.json $(MODULE_BINARY)
ifeq ($(VIAM_TARGET_OS), windows)
	jq '.entrypoint = "./bin/triangle-finder.exe"' meta.json > temp.json && mv temp.json meta.json
else
	strip $(MODULE_BINARY)
endif
	tar czf $@ meta.json $(MODULE_BINARY)
ifeq ($(VIAM_TARGET_OS), windows)
	git checkout meta.json
endif

test: setup
	GO_BUILD_ENV += GOOS=windows GOARCH=amd64
	GO_BUILD_FLAGS := -tags no_cgo	
	MODULE_BINARY = bin/viam-hough-transform.ex
	go test -v ./triangle_on_sonar_finder -run TestTriangleOnSonarFinde

module: module.tar.gz

setup:
	go mod tidy

