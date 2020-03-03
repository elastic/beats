PACKAGE = github.com/cavaliercoder/go-rpm

all: check install

check:
	go test -v $(PACKAGE)/...

install:
	go install -x $(PACKAGE)/...

clean: clean-fuzz
	go clean -x -i $(PACKAGE)/...

get-deps:
	go get github.com/cavaliercoder/badio
	go get github.com/cavaliercoder/go-cpio
	go get golang.org/x/crypto/openpgp

rpm-fuzz.zip: *.go
	go-fuzz-build $(PACKAGE)

fuzz: rpm-fuzz.zip
	go-fuzz -bin=./rpm-fuzz.zip -workdir=.fuzz/

clean-fuzz:
	rm -rf rpm-fuzz.zip .fuzz/crashers/* .fuzz/suppressions/*

get-fuzz-deps:
	go get github.com/dvyukov/go-fuzz/go-fuzz-build
	go get github.com/dvyukov/go-fuzz/go-fuzz

.PHONY: all check install clean get-deps fuzz clean-fuzz get-fuzz-deps
