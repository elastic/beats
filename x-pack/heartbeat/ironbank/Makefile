export BASE_REGISTRY := docker.elastic.co
export BASE_IMAGE := ubi8/ubi
export BASE_TAG := latest
BEAT_VERSION ?= 8.5.0

LOCATION := ../build/heartbeat-ironbank-$(BEAT_VERSION)-docker-build-context

JQ_URL = https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64
JQ_SHA256 = $(shell sha256sum $(LOCATION)/jq|cut -d " " -f 1)
NODE_VERSION = 12.22.5
NODE_URL = https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.gz
NODE_SHA256 = $(shell sha256sum $(LOCATION)/node.tar.gz|cut -d " " -f 1)
SYNTHETICS_VERSION = 1.0.0-beta.17
SYNTHETIC_URL = https://storage.googleapis.com/obs-ci-cache/beats/elastic-synthetics-$(SYNTHETICS_VERSION).tgz
SYNTHETIC_SHA256 = $(shell sha256sum $(LOCATION)/elastic-synthetics.tar.gz|cut -d " " -f 1)
TINIT_URL = https://github.com/krallin/tini/releases/download/v0.19.0/tini-amd64
TINIT_SHA256 = $(shell sha256sum $(LOCATION)/tinit|cut -d " " -f 1)

.PHONY: prepare
prepare: ## Download container dependencies.
	curl -sSfL -o $(LOCATION)/tinit $(TINIT_URL)
	curl -sSfL -o $(LOCATION)/jq $(JQ_URL)
	curl -sSfL -o $(LOCATION)/node.tar.gz $(NODE_URL)
	curl -sSfL -o $(LOCATION)/elastic-synthetics.tar.gz $(SYNTHETIC_URL)
	cp ../build/distributions/heartbeat-$(BEAT_VERSION)-linux-x86_64.tar.gz $(LOCATION)/
	@for i in $(shell cat rpm-deps.txt);\
	do \
		echo "Downloading $${i}"; \
		BASENAME=$$(basename "$${i}"); \
		curl -sSfL -o $(LOCATION)/$${BASENAME} "$${i}";\
	done

package: ## Package heartbeat for the artifacts consumed by the ironbank docker context.
	cd ../ ; \
		PLATFORMS='linux/amd64' PACKAGES=tar.gz mage package

.PHONY: validate-ironbank
validate-ironbank: prepare
	docker build \
		--build-arg BASE_REGISTRY=$(BASE_REGISTRY) \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg BASE_TAG=$(BASE_TAG) \
		$(LOCATION)
