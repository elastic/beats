BEATNAME=winlogbeat
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false

include scripts/Makefile

.PHONY: gen
gen: 
	GOOS=windows GOARCH=386 godep go generate -v -x ./...

# This is called by the beats-packer to obtain the configuration file and
# default template
.PHONY: install-cfg
install-cfg:
	cp etc/${BEATNAME}.template.json $(PREFIX)/${BEATNAME}.template.json
	# Windows
	cp etc/${BEATNAME}.yml $(PREFIX)/${BEATNAME}-win.yml
