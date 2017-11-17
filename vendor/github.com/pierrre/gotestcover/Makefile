#/bin/bash

setup:
	go get -u -v github.com/golang/lint/golint
	go get -v -t ./...

check:
	gofmt -d .
	go tool vet .
	golint

coverage:
	gotestcover -coverprofile=coverage.txt github.com/pierrre/gotestcover
	go tool cover -html=coverage.txt -o=coverage.html
	
clean:
	-rm coverage.txt
	-rm coverage.html
	gofmt -w .