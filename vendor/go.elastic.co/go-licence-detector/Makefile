.PHONY: generate
generate:
	@ go generate ./...
	@ go list -m -json all | go run main.go -noticeOut=NOTICE

.PHONY: build
build: generate
	@ go build -o bin/go-licence-detector

