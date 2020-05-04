TEST_TIMEOUT?=5m
GO_LICENSER_EXCLUDE=stacktrace/testdata

.PHONY: check
check: precheck check-modules test

.PHONY: precheck
precheck: check-goimports check-lint check-vet check-dockerfile-testing check-licenses

.PHONY: check-goimports
.PHONY: check-dockerfile-testing
.PHONY: check-lint
.PHONY: check-licenses
.PHONY: check-modules
ifeq ($(shell go run ./scripts/mingoversion.go -print 1.12),true)
check-goimports:
	sh scripts/check_goimports.sh

check-dockerfile-testing:
	go run ./scripts/gendockerfile.go -d

check-lint:
	sh scripts/check_lint.sh

check-licenses:
	go-licenser -d $(patsubst %,-exclude %,$(GO_LICENSER_EXCLUDE)) .

check-modules:
	go run scripts/genmod/main.go -check .
else
check-goimports:
check-dockerfile-testing:
check-lint:
check-licenses:
check-modules:
endif

.PHONY: check-vet
check-vet:
	@for dir in $(shell scripts/moduledirs.sh); do (cd $$dir && go vet ./...) || exit $$?; done

.PHONY: install
install:
	go get -v -t ./...

.PHONY: docker-test
docker-test:
	scripts/docker-compose-testing run -T --rm go-agent-tests make test

.PHONY: test
test:
	@for dir in $(shell scripts/moduledirs.sh); do (cd $$dir && go test -v -timeout=$(TEST_TIMEOUT) ./...) || exit $$?; done

.PHONY: coverage
coverage:
	@bash scripts/test_coverage.sh

.PHONY: fmt
fmt:
	@GOIMPORTSFLAGS=-w sh scripts/goimports.sh

.PHONY: clean
clean:
	rm -fr docs/html

.PHONY: update-modules
update-modules:
	go run scripts/genmod/main.go .

.PHONY: docs
docs:
ifdef ELASTIC_DOCS
	$(ELASTIC_DOCS)/build_docs --direct_html --chunk=1 $(BUILD_DOCS_ARGS) --doc docs/index.asciidoc --out docs/html
else
	@echo "\nELASTIC_DOCS is not defined.\n"
	@exit 1
endif

.PHONY: update-licenses
update-licenses:
	go-licenser $(patsubst %, -exclude %, $(GO_LICENSER_EXCLUDE)) .
