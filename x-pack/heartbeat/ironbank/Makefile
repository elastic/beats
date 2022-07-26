BUILD = build
NODE_VERSION = 12.22.5
NODE_URL = https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.gz

SYNTHETICS_VERSION = 1.0.0-beta.17
SYNTHETIC_URL = https://storage.googleapis.com/obs-ci-cache/beats/elastic-synthetics-$(SYNTHETICS_VERSION).tgz

.PHONY: generate-custom
generate-custom: ## Generate the custom hardening_manifest.
	rm -rf $(BUILD) || true
	mkdir -p $(BUILD)
	@for i in $(shell cat rpm-deps.txt);\
	do \
		BASENAME=$$(basename "$${i}"); \
		$(MAKE) download-and-add-entry URL=$${i} FILENAME=$${BASENAME} NAME=$${BASENAME}; \
	done
	$(MAKE) download-and-add-entry URL=$(NODE_URL) FILENAME=node.tar.gz NAME=node.tar.gz
	$(MAKE) download-and-add-entry URL=$(SYNTHETIC_URL) FILENAME=elastic-synthetics.tar.gz NAME=elastic-synthetics.tar.gz

download-and-add-entry:
	@echo "Downloading $(URL)"
	@curl -sSfL -o $(BUILD)/$(FILENAME) "$(URL)"
	@echo "  - filename: $(NAME)" 			>> $(BUILD)/hardening_manifest.yml.add
	@echo "    url: $(URL)" 				>> $(BUILD)/hardening_manifest.yml.add
	@echo "    validation:" 				>> $(BUILD)/hardening_manifest.yml.add
	@echo "      type: sha256" 				>> $(BUILD)/hardening_manifest.yml.add
	@echo "      value: $$(sha256sum $(BUILD)/$(FILENAME)|cut -d " " -f 1)" >> $(BUILD)/hardening_manifest.yml.add
