test:
	go test -v ./...

test-examples:
	@for example in $(shell find examples/ -name '*.go'); do \
		go build -v $$example || exit 1; \
	done
