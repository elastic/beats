BEAT_VERSION ?= 8.5.0

LOCATION := ../build/heartbeat-ironbank-$(BEAT_VERSION)-docker-build-context

.PHONY: download-hardening-manifest-artifacts
download-hardening-manifest-artifacts: ## Parse hardening_manifest.yaml and download each artifact.
	@for i in $(shell cat $(LOCATION)/hardening_manifest.yaml | grep ' url:' | grep 'https' | cut -d':' -f2-);\
	do \
		echo "Downloading $${i}"; \
		BASENAME=$$(basename "$${i}"); \
		curl -sSfL -o $(LOCATION)/$${BASENAME} "$${i}"; \
	done

.PHONY: prepare
prepare: download-hardening-manifest-artifacts ## Download container dependencies.
	cp ../build/distributions/heartbeat-$(BEAT_VERSION)-linux-x86_64.tar.gz $(LOCATION)/

package: ## Package heartbeat for the artifacts consumed by the ironbank docker context.
	cd ../ ; \
		PLATFORMS='linux/amd64' PACKAGES=tar.gz mage package

.PHONY: validate-ironbank
validate-ironbank: prepare
	docker build \
		--build-arg BASE_REGISTRY=docker.elastic.co \
		--build-arg BASE_IMAGE=ubi8/ubi \
		--build-arg BASE_TAG=latest \
		$(LOCATION)
