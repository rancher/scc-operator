# Include logic that can be reused across projects.
include hack/make/build.mk

# Define target platforms, image builder and the fully qualified image name.
TARGET_PLATFORMS ?= linux/amd64,linux/arm64

REPO ?= rancher
IMAGE ?= scc-operator
IMAGE_NAME = $(REPO)/$(IMAGE)
FULL_IMAGE_TAG = $(IMAGE_NAME):$(TAG)
BUILD_ACTION = --load

TARGETS := $(shell ls scripts)

$(TARGETS):
	./scripts/$@

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)

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

validate: validate-dirty ## Run validation checks.

validate-dirty:
ifdef DIRTY
	@echo Git is dirty
	@git --no-pager status
	@git --no-pager diff
	@exit 1
endif
