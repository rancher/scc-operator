# Include logic that can be reused across projects.
include hack/make/build.mk

# ---- CI Image Config ----
CI_IMAGE := ghcr.io/rancher/ci-image/go1.25
WORKDIR := /workspace

# Detect CI environment (common env var used by many CI systems)
CI ?= false

# Docker run wrapper (only used locally)
DOCKER_RUN = docker run --rm -i \
	-v $(PWD):$(WORKDIR) \
	-w $(WORKDIR) \
	$(CI_IMAGE)

# Command runner:
# - In CI: run commands directly
# - Locally: run via Docker
ifeq ($(CI),true)
	RUN =
else
	RUN = $(DOCKER_RUN)
endif

# ---- Build Config ----
# Define target platforms, image builder and the fully qualified image name.
TARGET_PLATFORMS ?= linux/amd64,linux/arm64

REPO ?= rancher
IMAGE ?= scc-operator
IMAGE_NAME = $(REPO)/$(IMAGE)
FULL_IMAGE_TAG = $(IMAGE_NAME):$(TAG)
BUILD_ACTION = --load

TARGETS := $(shell ls scripts)

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)
$(TARGETS):
	$(RUN) ./scripts/$@

clean: ## clean up project.
	rm -rf bin
	rm -rf dist
	rm -rf multiarch-image.oci
	rm -rf ci

build-image: buildx-machine ## build (and load) the container image targeting the current platform.
	$(IMAGE_BUILDER) build -f package/Dockerfile \
		--builder $(MACHINE) $(IMAGE_ARGS) \
		--build-arg VERSION=$(VERSION) \
		--platform=$(TARGET_PLATFORMS) \
		-t "$(FULL_IMAGE_TAG)" $(BUILD_ACTION) .
	@echo "Built $(FULL_IMAGE_TAG)"

push-image: validate buildx-machine ## build the container image targeting all platforms defined by TARGET_PLATFORMS and push to a registry.
	$(IMAGE_BUILDER) build -f package/Dockerfile \
		--builder $(MACHINE) $(IMAGE_ARGS) $(IID_FILE_FLAG) $(BUILDX_ARGS) \
		--build-arg VERSION=$(VERSION) \
		--platform=$(TARGET_PLATFORMS) \
		-t "$(FULL_IMAGE_TAG)" --push .
	@echo "Pushed $(FULL_IMAGE_TAG)"

.PHONY: validate
validate: validate-dirty ## Run validation checks.

.PHONY: validate-dirty
validate-dirty:
ifdef DIRTY
	@echo Git is dirty
	@git --no-pager status
	@git --no-pager diff
	@exit 1
endif
