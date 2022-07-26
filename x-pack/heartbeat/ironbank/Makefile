export BASE_REGISTRY := docker.elastic.co
export BASE_IMAGE := ubi8/ubi
export BASE_TAG := latest
export STACK_VERSION = 8.3.2  ## TODO To be replace with the current version

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

## TO BE DONE
prepare:
	@echo "As long as mage package has been launched"
	@echo "Then copy all the files to a particular location"
	@echo "And run the command make build within the new location"
	@echo "and make test."
	@echo "new locaiton should be ironbank complaince."

.PHONY: build
build: prepare
	docker-compose build

.PHONY: test
test: ## Test the container by launching a docker-compose with the Elastic Stack.
	STACK_VERSION=$(STACK_VERSION) \
	docker-compose up --no-start
	docker-compose start
	sleep 10
	docker-compose exec beat curl -I -f --max-time 5 --unix-socket '/usr/share/heartbeat/data/heartbeat.sock' 'http:/stats/?pretty'
