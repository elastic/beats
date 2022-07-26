BUILD = build
NODE_VERSION = 12.22.5
NODE_URL = https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.gz
NODE_SHA256 = $(shell sha256sum $(BUILD)/node.tar.gz|cut -d " " -f 1)
# npm view @elastic/synthetics dist.tarball
# SYNTHETIC_URL = https://registry.npmjs.org/@elastic/synthetics/-/synthetics-1.0.0-beta.30.tgz
# the npm package should be installed offline
SYNTHETICS_VERSION = 1.0.0-beta.17
SYNTHETIC_URL = https://storage.googleapis.com/obs-ci-cache/beats/elastic-synthetics-$(SYNTHETICS_VERSION).tgz
SYNTHETIC_SHA256 = $(shell sha256sum $(BUILD)/elastic-synthetics.tar.gz|cut -d " " -f 1)

.PHONY: generate-custom
generate-custom: ## Generate the custom hardening_manifest.
	rm -rf $(BUILD) || true
	mkdir -p $(BUILD)
	@for i in $(shell make dependencies);\
	do \
		echo "Downloading $${i}"; \
		BASENAME=$$(basename "$${i}"); \
		curl -sSfL -o $(BUILD)/$${BASENAME} "$${i}";\
		echo "  - filename: $${BASENAME}" >> $(BUILD)/hardening_manifest.yml.add; \
		echo "    url: $${i}" >> $(BUILD)/hardening_manifest.yml.add; \
		echo "    validation:" >> $(BUILD)/hardening_manifest.yml.add; \
		echo "      type: sha256" >> $(BUILD)/hardening_manifest.yml.add; \
		echo "      value: $$( sha256sum $(BUILD)/$${BASENAME}|cut -d " " -f 1)" >> $(BUILD)/hardening_manifest.yml.add; \
	done

dependencies: ## List the dependencies to be downloaded
	@echo $(NODE_URL)
	@echo $(SYNTHETIC_URL)
	@cat rpm-deps.txt
