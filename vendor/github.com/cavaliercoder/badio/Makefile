all: test lint

test:
	go test -v -cover

lint:
	gofmt -l -e -s . || :
	go vet . || :
	golint . || :
	gocyclo -over 15 . || :
	misspell ./* || :