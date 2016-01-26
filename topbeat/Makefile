BEATNAME=topbeat
SYSTEM_TESTS=true

# Only crosscompile for linux because other OS'es use cgo.
GOX_OS=linux

include ../libbeat/scripts/Makefile

.PHONY: install-cfg
install-cfg:
	cp etc/topbeat.template.json $(PREFIX)/topbeat.template.json
	# linux
	cp topbeat.yml $(PREFIX)/topbeat-linux.yml
	# binary
	cp topbeat.yml $(PREFIX)/topbeat-binary.yml
	# darwin
	cp topbeat.yml $(PREFIX)/topbeat-darwin.yml
	# win
	cp topbeat.yml $(PREFIX)/topbeat-win.yml

