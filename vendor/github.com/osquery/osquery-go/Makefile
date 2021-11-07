PATH := $(GOPATH)/bin:$(PATH)
export GO111MODULE=on

all: gen examples

go-mod-check:
	@go help mod > /dev/null || (echo "Your go is too old, no modules. Seek help." && exit 1)

go-mod-download:
	go mod download

deps-go: go-mod-check go-mod-download

deps: deps-go

gen: ./osquery.thrift
	mkdir -p ./gen
	thrift --gen go:package_prefix=github.com/osquery/osquery-go/gen/ -out ./gen ./osquery.thrift
	rm -rf gen/osquery/extension-remote gen/osquery/extension_manager-remote
	gofmt -w ./gen

examples: example_query example_call example_logger example_distributed example_table example_config

example_query: examples/query/*.go
	go build -o example_query ./examples/query/*.go

example_call: examples/call/*.go
	go build -o example_call ./examples/call/*.go

example_logger: examples/logger/*.go
	go build -o example_logger.ext  ./examples/logger/*.go

example_distributed: examples/distributed/*.go
	go build -o example_distributed.ext  ./examples/distributed/*.go

example_table: examples/table/*.go
	go build -o example_table ./examples/table/*.go

example_config: examples/config/*.go
	go build -o example_config ./examples/config/*.go

test: all
	go test -race -cover ./...

clean:
	rm -rf ./build ./gen

.PHONY: all
