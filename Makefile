.PHONY: default all build release
THIS_FILE := $(lastword $(MAKEFILE_LIST))

APP ?= spang
ifdef SPANG_VERSION
	BUILD_VERSION := $(SPANG_VERSION)
endif
ifdef DOCKER_REGISTRY_ADDR
	REGISTRY := $(DOCKER_REGISTRY_ADDR)
endif
REGISTRY ?= registry.hub.docker.com

BUILD_VERSION ?= $(shell git describe --always --dirty)
TARGET_IMAGE := $(REGISTRY)/$(DOCKER_REGISTRY_USER)/$(APP):$(BUILD_VERSION)
export APP BUILD_VERSION TARGET_IMAGE

BUILD_FLAGS := --build-arg APP=$(APP) --build-arg BUILD_VERSION=$(BUILD_VERSION) --build-arg BUILD_ID=$(BUILD_ID) --build-arg TARGET_IMAGE=$(TARGET_IMAGE)
RUN_BUILD := docker build $(BUILD_FLAGS) -t $(TARGET_IMAGE) .

default: build

all: build

build:
	@echo $@
	$(RUN_BUILD)

container: build
	@echo $@
	@$(MAKE) -f $(THIS_FILE) clean

release: container
	@echo $@
	@echo "Publishing image..."
	docker login -u $(DOCKER_REGISTRY_USER) -p $(DOCKER_REGISTRY_PASSPHRASE) -e nomail $(REGISTRY)
	docker push $(TARGET_IMAGE)
