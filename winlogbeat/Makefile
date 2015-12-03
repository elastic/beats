BEATNAME=winlogbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
GOX_OS=windows

include ../libbeat/scripts/Makefile

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 godep go generate -v -x ./...

.PHONY: gen-docs
gen-docs:
	python scripts/generate_field_docs.py etc/fields.yml docs/fields.asciidoc

# This is called by the beats-packer to obtain the configuration file and
# default template
.PHONY: install-cfg
install-cfg:
	cp etc/$(BEATNAME).template.json $(PREFIX)/$(BEATNAME).template.json
	# Windows
	cp etc/${BEATNAME}.yml $(PREFIX)/${BEATNAME}-win.yml
	sed -i 's|#\{0,1\}\(registry_file:\).*|\1 C:/ProgramData/winlogbeat/.winlogbeat.yaml|' $(PREFIX)/$(BEATNAME)-win.yml
	sed -i 's|#\{0,1\}\(to_files:\).*|\1 true|' $(PREFIX)/$(BEATNAME)-win.yml
	sed -i 's|#\{0,1\}\(level:\).*|\1 info|' $(PREFIX)/$(BEATNAME)-win.yml
	sed -i '/log files/{n;s|#\{0,1\}path:.*|path: C:/ProgramData/winlogbeat/Logs|}' $(PREFIX)/$(BEATNAME)-win.yml
