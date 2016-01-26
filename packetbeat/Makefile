
BEATNAME=packetbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false

include ../libbeat/scripts/Makefile

.PHONY: with_pfring
with_pfring:
	go build --tags havepfring


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



.PHONY: benchmark
benchmark:
	$(GODEP) go test -short -bench=. ./... -cpu=2

.PHONY: gen
gen: python-env
	. ${PYTHON_ENV}/bin/activate && python scripts/generate_template.py   etc/fields.yml etc/packetbeat.template.json
	. ${PYTHON_ENV}/bin/activate && python scripts/generate_field_docs.py etc/fields.yml docs/fields.asciidoc
