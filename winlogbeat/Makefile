BEATNAME=winlogbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false
GOX_OS=windows

include ../libbeat/scripts/Makefile

.PHONY: gen
gen:
	GOOS=windows GOARCH=386 go generate -v -x ./...

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
	# Windows
	sed -i 's|#\{0,1\}\(to_files:\).*|\1 true|' $(PREFIX)/$(BEATNAME)-win.yml
	sed -i 's|#\{0,1\}\(level:\).*|\1 info|' $(PREFIX)/$(BEATNAME)-win.yml
	sed -i '/log files/{n;s|#\{0,1\}path:.*|path: C:/ProgramData/winlogbeat/Logs|}' $(PREFIX)/$(BEATNAME)-win.yml
