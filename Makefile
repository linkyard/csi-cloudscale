NAME=cloudscale-csi-plugin
OS ?= linux
ifeq ($(strip $(shell git status --porcelain 2>/dev/null)),)
  GIT_TREE_STATE=clean
else
  GIT_TREE_STATE=dirty
endif
COMMIT ?= $(shell git rev-parse HEAD)
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
LDFLAGS ?= -X github.com/cloudscale-ch/csi-cloudscale/driver.version=${VERSION} -X github.com/cloudscale-ch/csi-cloudscale/driver.commit=${COMMIT} -X github.com/cloudscale-ch/csi-cloudscale/driver.gitTreeState=${GIT_TREE_STATE}
PKG ?= github.com/cloudscale-ch/csi-cloudscale/cmd/cloudscale-csi-plugin

VERSION ?= $(shell git describe)

all: test

.PHONY: compile
compile:
	@echo "==> Building the project v${VERSION}"
	@env CGO_ENABLED=0 GOOS=${OS} GOARCH=amd64 go build -o cmd/cloudscale-csi-plugin/${NAME} -ldflags "$(LDFLAGS)" ${PKG}

.PHONY: test
test:
	@echo "==> Testing all packages"
	@go test -v ./...

.PHONY: test-integration
test-integration:
	@echo "==> Started integration tests"
	@env GOCACHE=off go test -v -tags integration ./test/...

.PHONY: build
build: compile
	@echo "==> Building the docker image"
	@docker build -t cloudscalech/cloudscale-csi-plugin:$(VERSION) -f cmd/cloudscale-csi-plugin/Dockerfile cmd/cloudscale-csi-plugin

.PHONY: clean
clean:
	@echo "==> Cleaning releases"
	@GOOS=${OS} go clean -i -x ./...
