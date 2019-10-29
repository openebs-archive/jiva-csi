# Output registry and image names for operator image
# Set env to override this value
REGISTRY ?= openebs

# Output plugin name and its image name and tag
PLUGIN_NAME=jiva-csi
PLUGIN_TAG=ci

GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD | sed -e "s/.*\\///")
GIT_TAG = $(shell git describe --tags)

# use git branch as default version if not set by env variable, if HEAD is detached that use the most recent tag
VERSION ?= $(if $(subst HEAD,,${GIT_BRANCH}),$(GIT_BRANCH),$(GIT_TAG))
COMMIT ?= $(shell git rev-parse HEAD | cut -c 1-7)
ifeq ($(GIT_TAG),)
	GIT_TAG := $(COMMIT)
endif
DATETIME ?= $(shell date +'%F_%T')
LDFLAGS ?= \
        -extldflags "-static" \
	-X github.com/openebs/jiva-csi/version/version.Version=${VERSION} \
	-X github.com/openebs/jiva-csi/version/version.Commit=${COMMIT} \
	-X github.com/openebs/jiva-csi/version/version.DateTime=${DATETIME}

# list only csi source code directories
PACKAGES = $(shell go list ./... | grep -v 'vendor')

.PHONY: all
all:
	@echo "Available commands:"
	@echo "  build                           - build csi source code"
	@echo "  image                           - build csi container image"
	@echo "  push                            - push csi to dockerhub registry (${REGISTRY})"
	@echo ""
	@make print-variables --no-print-directory

.PHONY: print-variables
print-variables:
	@echo "Variables:"
	@echo "  VERSION:    ${VERSION}"
	@echo "  GIT_BRANCH: ${GIT_BRANCH}"
	@echo "  GIT_TAG:    ${GIT_TAG}"
	@echo "  COMMIT:     ${COMMIT}"
	@echo "Testing variables:"
	@echo " Produced Image: ${PLUGIN_NAME}:${PLUGIN_TAG}"
	@echo " REGISTRY: ${REGISTRY}"


.get:
	rm -rf ./build/bin/
	GO111MODULE=on go mod download

deps: .get
	GO111MODULE=on go mod vendor

build: deps test
	GO111MODULE=on GOOS=linux go build -a -ldflags '$(LDFLAGS)' -o ./build/bin/$(PLUGIN_NAME) ./cmd/csi/main.go

image: build
	docker build -f ./build/Dockerfile -t $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) .

push: image
	docker push $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG)

tag:
	docker tag $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) $(REGISTRY)/$(PLUGIN_NAME):$(GIT_TAG)

push-tag: tag
	docker push $(REGISTRY)/$(PLUGIN_NAME):$(GIT_TAG)

clean:
	rm -rf ./build/bin/

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

test: format
	@echo "--> Running go test" ;
	@go test $(PACKAGES)

