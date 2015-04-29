.PHONY: build
build: 
	go build ./...

.PHONY: deps
deps:
	go get -t ./...
	# goautotest is used from the Makefile to run tests in a loop
	go get github.com/tsg/goautotest
	# cover
	go get golang.org/x/tools/cmd/cover

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: test
test:
	go test -short ./...

.PHONY: autotest
autotest:
	goautotest -short ./...

.PHONY: testlong
testlong:
	go vet ./...
	go test ./...

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./...

.PHONY: cover
cover:
	mkdir -p cover
	./scripts/coverage.sh
	go tool cover -html=profile.cov -o cover/coverage.html
