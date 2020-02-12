##@ Test
test-unit:  ## Run unit tests
	@echo "\033[2m→ Running unit tests...\033[0m"
ifdef race
	$(eval testunitargs += "-race")
endif
	$(eval testunitargs += "-cover" "-coverprofile=tmp/unit.cov" "./...")
	@mkdir -p tmp
	@if which gotestsum > /dev/null 2>&1 ; then \
		echo "gotestsum --format=short-verbose --junitfile=tmp/unit-report.xml --" $(testunitargs); \
		gotestsum --format=short-verbose --junitfile=tmp/unit-report.xml -- $(testunitargs); \
	else \
		echo "go test -v" $(testunitargs); \
		go test -v $(testunitargs); \
	fi;
test: test-unit

test-integ:  ## Run integration tests
	@echo "\033[2m→ Running integration tests...\033[0m"
ifdef race
	$(eval testintegargs += "-race")
endif
	$(eval testintegargs += "-cover" "-coverprofile=tmp/integration-client.cov" "-tags='integration'" "-timeout=1h" "github.com/elastic/go-elasticsearch" "github.com/elastic/go-elasticsearch/estransport")
	@mkdir -p tmp
	@if which gotestsum > /dev/null 2>&1 ; then \
		echo "gotestsum --format=short-verbose --junitfile=tmp/integration-report.xml --" $(testintegargs); \
		gotestsum --format=short-verbose --junitfile=tmp/integration-report.xml -- $(testintegargs); \
	else \
		echo "go test -v" $(testintegargs); \
		go test -v $(testintegargs); \
	fi;

test-api:  ## Run generated API integration tests
	@echo "\033[2m→ Running API integration tests...\033[0m"
ifdef race
	$(eval testapiargs += "-race")
endif
	$(eval testapiargs += "-cover" "-coverpkg=github.com/elastic/go-elasticsearch/esapi" "-coverprofile=$(PWD)/tmp/integration-api.cov" "-tags='integration'" "-timeout=1h" "./...")
	@mkdir -p tmp
	@if which gotestsum > /dev/null 2>&1 ; then \
		echo "cd esapi/test && gotestsum --format=short-verbose --junitfile=$(PWD)/tmp/integration-api-report.xml --" $(testapiargs); \
		cd esapi/test && gotestsum --format=short-verbose --junitfile=$(PWD)/tmp/integration-api-report.xml -- $(testapiargs); \
	else \
		echo "go test -v" $(testapiargs); \
		cd esapi/test && go test -v $(testapiargs); \
	fi;

test-bench:  ## Run benchmarks
	@echo "\033[2m→ Running benchmarks...\033[0m"
	go test -run=none -bench=. -benchmem ./...

test-examples: ## Execute the _examples
	@echo "\033[2m→ Testing the examples...\033[0m"
	@{ \
		set -e ; \
		for f in _examples/*.go; do \
			echo "\033[2m────────────────────────────────────────────────────────────────────────────────"; \
			echo "\033[1m$$f\033[0m"; \
			echo "\033[2m────────────────────────────────────────────────────────────────────────────────\033[0m"; \
			(go run $$f && true) || \
			( \
				echo "\033[31m────────────────────────────────────────────────────────────────────────────────\033[0m"; \
				echo "\033[31;1m⨯ ERROR\033[0m"; \
				false; \
			); \
		done; \
		\
		for f in _examples/*/; do \
			echo "\033[2m────────────────────────────────────────────────────────────────────────────────\033[0m"; \
			echo "\033[1m$$f\033[0m"; \
			echo "\033[2m────────────────────────────────────────────────────────────────────────────────\033[0m"; \
			(cd $$f && make test && true) || \
			( \
				echo "\033[31m────────────────────────────────────────────────────────────────────────────────\033[0m"; \
				echo "\033[31;1m⨯ ERROR\033[0m"; \
				false; \
			); \
		done; \
		echo "\033[32m────────────────────────────────────────────────────────────────────────────────\033[0m"; \
		\
		echo "\033[32;1mSUCCESS\033[0m"; \
	}

test-coverage:  ## Generate test coverage report
	@echo "\033[2m→ Generating test coverage report...\033[0m"
	@go tool cover -html=tmp/unit.cov -o tmp/coverage.html
	@go tool cover -func=tmp/unit.cov | 'grep' -v 'esapi/api\.' | sed 's/github.com\/elastic\/go-elasticsearch\///g'
	@echo "--------------------------------------------------------------------------------\nopen tmp/coverage.html\n"

##@ Development
lint:  ## Run lint on the package
	@echo "\033[2m→ Running lint...\033[0m"
	go vet github.com/elastic/go-elasticsearch/...
	go list github.com/elastic/go-elasticsearch/... | 'grep' -v internal | xargs golint -set_exit_status

apidiff: ## Display API incompabilities
	@if ! command -v apidiff > /dev/null; then \
		echo "\033[31;1mERROR: apidiff not installed\033[0m"; \
		echo "go get -u github.com/go-modules-by-example/apidiff"; \
		echo "\033[2m→ https://github.com/go-modules-by-example/index/blob/master/019_apidiff/README.md\033[0m\n"; \
		false; \
	fi;
	@rm -rf tmp/apidiff-OLD tmp/apidiff-NEW
	@git clone --quiet --local .git/ tmp/apidiff-OLD
	@mkdir -p tmp/apidiff-NEW
	@tar -c --exclude .git --exclude tmp --exclude cmd . | tar -x -C tmp/apidiff-NEW
	@echo "\033[2m→ Running apidiff...\033[0m"
	@echo "tmp/apidiff-OLD/esapi tmp/apidiff-NEW/esapi"
	@{ \
		set -e ; \
		output=$$(apidiff tmp/apidiff-OLD/esapi tmp/apidiff-NEW/esapi); \
		echo "\n$$output\n"; \
		if echo $$output | grep -e 'incompatible' -; then \
			echo "\n\033[31;1mFAILURE\033[0m\n"; \
			false; \
		else \
			echo "\033[32;1mSUCCESS\033[0m"; \
		fi; \
	}

godoc: ## Display documentation for the package
	@echo "\033[2m→ Generating documentation...\033[0m"
	@echo "open http://localhost:6060/pkg/github.com/elastic/go-elasticsearch/\n"
	mkdir -p /tmp/tmpgoroot/doc
	rm -rf /tmp/tmpgopath/src/github.com/elastic/go-elasticsearch
	mkdir -p /tmp/tmpgopath/src/github.com/elastic/go-elasticsearch
	tar -c --exclude='.git' --exclude='tmp' . | tar -x -C /tmp/tmpgopath/src/github.com/elastic/go-elasticsearch
	GOROOT=/tmp/tmpgoroot/ GOPATH=/tmp/tmpgopath/ godoc -http=localhost:6060 -play

cluster: ## Launch an Elasticsearch cluster with Docker
	$(eval version ?= "elasticsearch-oss:7.0.0-SNAPSHOT")
ifeq ($(origin nodes), undefined)
	$(eval nodes = 1)
endif
	@echo "\033[2m→ Launching" $(nodes) "node(s) of" $(version) "...\033[0m"
ifeq ($(shell test $(nodes) && test $(nodes) -gt 1; echo $$?),0)
	$(eval detached ?= "true")
else
	$(eval detached ?= "false")
endif
	@docker network inspect elasticsearch > /dev/null || docker network create elasticsearch;
	@{ \
		for n in `seq 1 $(nodes)`; do \
			docker run \
				--name "es$$n" \
				--network elasticsearch \
				--env "node.name=es$$n" \
				--env "cluster.name=go-elasticsearch" \
				--env "cluster.initial_master_nodes=es1" \
				--env "cluster.routing.allocation.disk.threshold_enabled=false" \
				--env "discovery.zen.ping.unicast.hosts=es1" \
				--env "bootstrap.memory_lock=true" \
				--env "node.attr.testattr=test" \
				--env "path.repo=/tmp" \
				--env "repositories.url.allowed_urls=http://snapshot.test*" \
				--env ES_JAVA_OPTS="-Xms1g -Xmx1g" \
				--volume es$$n-data:/usr/share/elasticsearch/data \
				--publish $$((9199+$$n)):9200 \
				--ulimit nofile=65536:65536 \
				--ulimit memlock=-1:-1 \
				--detach=$(detached) \
				--rm \
				docker.elastic.co/elasticsearch/$(version); \
		done \
	}
	@{ \
		if [[ "$(detached)" == "true" ]]; then \
			echo "\033[2m→ Waiting for the cluster...\033[0m"; \
			docker run --network elasticsearch --rm appropriate/curl --max-time 120 --retry 120 --retry-delay 1 --retry-connrefused --show-error --silent http://es1:9200; \
		fi \
	}

docker: ## Build the Docker image and run it
	docker build --file Dockerfile --tag elastic/go-elasticsearch .
	docker run -it --network elasticsearch --volume $(PWD)/tmp:/tmp:rw,delegated --rm elastic/go-elasticsearch

##@ Generator
gen-api:  ## Generate the API package from the JSON specification
	@echo "\033[2m→ Generating API package from specification...\033[0m"
	$(eval input  ?= tmp/elasticsearch/rest-api-spec/src/main/resources/rest-api-spec/api/*.json)
	$(eval output ?= esapi)
ifdef debug
	$(eval args += --debug)
endif
ifdef skip-registry
	$(eval args += --skip-registry)
endif
	cd internal/cmd/generate && go run main.go source --input '$(PWD)/$(input)' --output '$(PWD)/$(output)' $(args)

gen-tests:  ## Generate the API tests from the YAML specification
	@echo "\033[2m→ Generating API tests from specification...\033[0m"
	$(eval input  ?= tmp/elasticsearch/rest-api-spec/src/main/resources/rest-api-spec/test/**/*.yml)
	$(eval output ?= esapi/test)
ifdef debug
	$(eval args += --debug)
endif
	rm -rf esapi/test/*_test.go
	cd internal/cmd/generate && go generate ./... && go run main.go tests --input '$(PWD)/$(input)' --output '$(PWD)/$(output)' $(args)

##@ Other
#------------------------------------------------------------------------------
help:  ## Display help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
#------------- <https://suva.sh/posts/well-documented-makefiles> --------------

.DEFAULT_GOAL := help
.PHONY: help apidiff coverage docker examples gen-api gen-tests godoc lint test test-api test-bench test-integ test-unit
