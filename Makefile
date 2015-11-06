GODEP=$(GOPATH)/bin/godep

# default install target used by the beats-packer
PREFIX?=/build

GOFILES = $(shell find . -type f -name '*.go')
packetbeat: $(GOFILES)
	# first make sure we have godep
	go get github.com/tools/godep
	$(GODEP) go build

.PHONY: with_pfring
with_pfring:
	go build --tags havepfring

.PHONY: getdeps
getdeps:
	go get -t -u -f ./...
	# goautotest is used from the Makefile to run tests in a loop
	go get -u github.com/tsg/goautotest
	# websocket is needed by the gobeacon tests
	go get -u golang.org/x/net/websocket
	# godep is needed in this makefile
	go get -u github.com/tools/godep
	# go-fuzz and go-fuzz-build are needed for fuzzy testing
	go get -u github.com/dvyukov/go-fuzz/go-fuzz
	go get -u github.com/dvyukov/go-fuzz/go-fuzz-build

.PHONY: updatedeps
updatedeps:
	$(GODEP) update ...

# This is called by the beats-packer to obtain the configuration file
.PHONY: install-cfg
install-cfg:
	cp etc/packetbeat.template.json $(PREFIX)/packetbeat.template.json
	# linux
	cp etc/packetbeat.yml $(PREFIX)/packetbeat-linux.yml
	# binary
	cp etc/packetbeat.yml $(PREFIX)/packetbeat-binary.yml
	# darwin
	cp etc/packetbeat.yml $(PREFIX)/packetbeat-darwin.yml
	sed -i.bk 's/device: any/device: en0/' $(PREFIX)/packetbeat-darwin.yml
	rm $(PREFIX)/packetbeat-darwin.yml.bk
	# win
	cp etc/packetbeat.yml $(PREFIX)/packetbeat-win.yml
	sed -i.bk 's/device: any/device: 0/' $(PREFIX)/packetbeat-win.yml
	rm $(PREFIX)/packetbeat-win.yml.bk

.PHONY: gofmt
gofmt:
	go fmt ./...

.PHONY: test
test:
	$(GODEP) go test -short ./...
	make -C tests test

.PHONY: autotest
autotest:
	goautotest -short ./...

.PHONY: testlong
testlong:
	go vet ./...
	make coverage
	make -C tests test

.PHONY: coverage
coverage:
	# gotestcover is needed to fetch coverage for multiple packages
	go get github.com/pierrre/gotestcover
	mkdir -p coverage
	GOPATH=$(shell $(GODEP) path):$(GOPATH) $(GOPATH)/bin/gotestcover -coverprofile=./coverage/unit.cov -covermode=count github.com/elastic/packetbeat/...
	$(GODEP) go tool cover -html=./coverage/unit.cov -o coverage/unit.html

.PHONY: benchmark
benchmark:
	$(GODEP) go test -short -bench=. ./... -cpu=2

.PHONY: testfuzz
testfuzz:
	mkdir -p go-fuzz/workdir/corpus
	$(GODEP) go build -o go-fuzz/workdir/prepare_corpus go-fuzz/prepare_corpus.go
	cd go-fuzz/workdir/corpus; find ../../../tests/pcaps/ -name '*.pcap' | xargs -n1 ../prepare_corpus
	find go-fuzz/workdir/corpus -type f -size 0 -exec rm {} \;
	# not possible to use together with godeb -> so have the correct version of all dependencies checked out (e.g. go-thrift)
	go-fuzz-build -o go-fuzz/workdir/decoder-fuzz.zip github.com/elastic/packetbeat/decoder
	nice go-fuzz -bin=go-fuzz/workdir/decoder-fuzz.zip -workdir=go-fuzz/workdir -dup -timeout 60

.PHONY: env
env: env/bin/activate
env/bin/activate: requirements.txt
	test -d env || virtualenv env > /dev/null
	. env/bin/activate && pip install -Ur requirements.txt > /dev/null
	touch env/bin/activate

.PHONY: gen
gen: env
	. env/bin/activate && python scripts/generate_template.py   etc/fields.yml etc/packetbeat.template.json
	. env/bin/activate && python scripts/generate_field_docs.py etc/fields.yml docs/fields.asciidoc

.PHONY: clean
clean:
	-rm packetbeat
	-rm packetbeat.test
	-rm -r coverage
	-rm -r env
	-rm -r go-fuzz/workdir

# Generates packetbeat.test coverage testing binary
packetbeat.test: $(GOFILES)
	$(GODEP) go test -c -cover -covermode=count -coverpkg ./...

full-coverage:
	make coverage
	make -C ./tests coverage
	# Writes count mode on top of file
	echo 'mode: count' > ./coverage/full.cov
	# Collects all coverage files and skips top line with mode
	tail -q -n +2 ./coverage/*.cov >> ./coverage/full.cov
	$(GODEP) go tool cover -html=./coverage/full.cov -o coverage/full.html
