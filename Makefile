# Output registry and image names for operator image
# Set env to override this value
REGISTRY ?= openebs

# Output plugin name and its image name and tag
PLUGIN_NAME=jiva-csi
PLUGIN_TAG=ci

# Tools required for different make targets or for development purposes
EXTERNAL_TOOLS=\
	golang.org/x/tools/cmd/cover \
	github.com/axw/gocov/gocov \
	github.com/ugorji/go/codec/codecgen \

# Lint our code. Reference: https://golang.org/cmd/vet/
VETARGS?=-asmdecl -atomic -bool -buildtags -copylocks -methods \
         -nilfunc -printf -rangeloops -shift -structtag -unsafeptr

GIT_BRANCH = $(shell git rev-parse --abbrev-ref HEAD | sed -e "s/.*\\///")
GIT_TAG = $(shell git describe --tags)

# use git branch as default version if not set by env variable, if HEAD is detached that use the most recent tag
VERSION ?= $(if $(subst HEAD,,${GIT_BRANCH}),$(GIT_BRANCH),$(GIT_TAG))
COMMIT ?= $(shell git rev-parse HEAD | cut -c 1-7)

ifeq ($(GIT_TAG),)
	GIT_TAG := $(COMMIT)
endif

ifeq (${TRAVIS_TAG}, )
  GIT_TAG = $(COMMIT)
	export GIT_TAG
else
  GIT_TAG = ${TRAVIS_TAG}
	export GIT_TAG
endif

PACKAGES = $(shell go list ./... | grep -v 'vendor')

DATETIME ?= $(shell date +'%F_%T')
LDFLAGS ?= \
        -extldflags "-static" \
	-X github.com/openebs/jiva-csi/version/version.Version=${VERSION} \
	-X github.com/openebs/jiva-csi/version/version.Commit=${COMMIT} \
	-X github.com/openebs/jiva-csi/version/version.DateTime=${DATETIME}


.PHONY: help
help:
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

# Bootstrap the build by downloading additional tools
bootstrap:
	@for tool in  $(EXTERNAL_TOOLS) ; do \
		echo "+ Installing $$tool" ; \
		go get -u $$tool; \
	done

.get:
	rm -rf ./build/bin/
	go mod download

vet:
	@go vet 2>/dev/null ; if [ $$? -eq 3 ]; then \
		go get golang.org/x/tools/cmd/vet; \
	fi
	@echo "--> Running go tool vet ..."
	@go vet $(VETARGS) ${PACKAGES} ; if [ $$? -eq 1 ]; then \
		echo ""; \
		echo "[LINT] Vet found suspicious constructs. Please check the reported constructs"; \
		echo "and fix them if necessary before submitting the code for review."; \
	fi

	@git grep -n `echo "log"".Print"` | grep -v 'vendor/' ; if [ $$? -eq 0 ]; then \
		echo "[LINT] Found "log"".Printf" calls. These should use Maya's logger instead."; \
	fi


deps: .get
	go mod vendor

build: deps test
	@echo "--> Build binary $(PLUGIN_NAME) ..."
	GOOS=linux go build -a -ldflags '$(LDFLAGS)' -o ./build/bin/$(PLUGIN_NAME) ./cmd/csi/main.go

image: build
	@echo "--> Build image $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) ..."
	docker build -f ./build/Dockerfile -t $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) .

push-image: image
	@echo "--> Push image $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) ..."
	docker push $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG)

push:
	@echo "--> Push image $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) ..."
	docker push $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG)

tag:
	@echo "--> Tag image $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) to $(REGISTRY)/$(PLUGIN_NAME):$(GIT_TAG) ..."
	docker tag $(REGISTRY)/$(PLUGIN_NAME):$(PLUGIN_TAG) $(REGISTRY)/$(PLUGIN_NAME):$(GIT_TAG)

push-tag: tag push
	@echo "--> Push image $(REGISTRY)/$(PLUGIN_NAME):$(GIT_TAG) ..."
	docker push $(REGISTRY)/$(PLUGIN_NAME):$(GIT_TAG)

clean:
	rm -rf ./build/bin/
	go mod tidy

format:
	@echo "--> Running go fmt"
	@go fmt $(PACKAGES)

test: format vet clean
	@echo "--> Running go test" ;
	@go test -v --cover $(PACKAGES)

