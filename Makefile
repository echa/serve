.PHONY: default all build release
THIS_FILE := $(lastword $(MAKEFILE_LIST))

ARTIFACT := serve
FLAVOR ?= alpine

ifdef SPANG_VERSION
	BUILD_VERSION := $(SPANG_VERSION)-$(FLAVOR)
endif
BUILD_VERSION ?= $(shell cat VERSION)-$(FLAVOR)
BUILD_VERSION ?= $(shell git describe --always --dirty)-$(FLAVOR)
BUILD_DATE := $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")
ifndef BUILD_ID
	BUILD_ID := $(shell uuid)
endif
ifdef DOCKER_REGISTRY_ADDR
	REGISTRY := $(DOCKER_REGISTRY_ADDR)
endif

# Uisng public docker registry
# REGISTRY ?= registry.hub.docker.com
# TARGET_IMAGE := $(REGISTRY)/$(DOCKER_REGISTRY_USER)/$(APP):$(BUILD_VERSION)

# using private registry
TARGET_IMAGE := $(REGISTRY)/$(ARTIFACT):$(BUILD_VERSION)

export ARTIFACT TARGET_IMAGE BUILD_ID BUILD_VERSION BUILD_DATE

default: build

all: build

build:
	@echo $@
	@docker build --pull --rm --no-cache --build-arg BUILD_DATE=$(BUILD_DATE) --build-arg BUILD_VERSION=$(BUILD_VERSION) --build-arg BUILD_ID=$(BUILD_ID) -t $(TARGET_IMAGE) .

image: | build clean
	@echo $@
	@echo
	@echo "Docker image complete. Continue with "
	@echo " List:         docker images"
	@echo " Push:         docker push $(TARGET_IMAGE)"
	@echo " Inspect:      docker inspect $(ARGET_IMAGE)"
	@echo " Run:          docker run --rm --name $(ARTIFACT) $(TARGET_IMAGE)"
	@echo

release: image
	@echo $@
	@echo "Publishing image..."
	docker login -u $(DOCKER_REGISTRY_USER) -p $(DOCKER_REGISTRY_PASSPHRASE) $(REGISTRY)
	docker push $(TARGET_IMAGE)

clean:
	@echo $@
	docker image prune --force --filter label=stage=serve-builder --filter label=build=$(BUILD_ID)
