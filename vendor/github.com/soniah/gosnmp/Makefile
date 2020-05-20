.PHONY: test lint lint-all lint-examples tools

test:
	go test *.go

# gradually build up amount of linting - there's a lot to do...

lint: lint-examples
	# start increasing linting level..
	golangci-lint run --disable-all -E goimports
	golangci-lint run ./...
	golangci-lint run -p bugs


lint-examples:
	# recursively lint the examples
	cd examples && golangci-lint run -p bugs -p complexity -p unused -p format -E lll -E interfacer

lint-all:
	# recursively lint all files, all commits - ugh
	# golangci-lint run -p bugs -p complexity -p unused -p format -E lll -E interfacer *.go

tools:
	# install build tools
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.24.0
