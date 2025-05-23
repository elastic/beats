BEAT_VERSION ?= 8.9.0
LOCATION := ../build/heartbeat-ironbank-$(BEAT_VERSION)-docker-build-context

YQ_VERSION ?= v4.13.2
YQ ?= "../build/yq"
## Support for 64 bits (arm and amd/intel)
ARCH = $(shell uname -m)
ifeq ($(ARCH),x86_64)
	YQ_ARCH ?= amd64
else
	YQ_ARCH ?= arm64
endif
OS = $(shell uname)
ifeq ($(OS),Darwin)
	YQ_BINARY ?= yq_darwin_$(YQ_ARCH)
else
	YQ_BINARY ?= yq_linux_$(YQ_ARCH)
endif

## @help:setup-yq: Install yq in ../build/yq.
.PHONY: setup-yq
setup-yq:
	@if [ ! -f "../build/yq" ]; then \
		echo "Downloading yq - $(YQ_VERSION)/$(YQ_BINARY)" ; \
		curl -sSfL -o $(YQ) https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/$(YQ_BINARY) ; \
		chmod +x $(YQ) ; \
	fi

.PHONY: download-hardening-manifest-artifacts
download-hardening-manifest-artifacts: setup-yq ## Parse hardening_manifest.yaml and download each artifact.
	@for i in $(shell $(YQ) -M e '.resources[] | select(.url == "https*") | { .filename: .url }' - < $(LOCATION)/hardening_manifest.yaml | sed 's#: #=#g');\
	do \
		BASENAME=$$(echo "$${i}" | cut -d'=' -f1); \
		URL=$$(echo "$${i}" | cut -d'=' -f2); \
		echo "Downloading $${URL}"; \
		curl -sSfL -o $(LOCATION)/$${BASENAME} "$${URL}"; \
	done

.PHONY: prepare
prepare: download-hardening-manifest-artifacts ## Download container dependencies.
	cp -f ../build/distributions/heartbeat-$(BEAT_VERSION)-linux-x86_64.tar.gz $(LOCATION)/

package: ## Package heartbeat for the artifacts consumed by the ironbank docker context.
	cd ../ ; \
		PLATFORMS='linux/amd64' PACKAGES=tar.gz mage package

.PHONY: validate-ironbank
validate-ironbank: prepare
	docker build \
		--build-arg BASE_REGISTRY=docker.elastic.co \
		--build-arg BASE_IMAGE=ubi9/ubi \
		--build-arg BASE_TAG=latest \
		$(LOCATION)
