BEATNAME=topbeat
SYSTEM_TESTS=true

# Only crosscompile for linux because other OS'es use cgo.
GOX_OS=linux

include ../libbeat/scripts/Makefile

.PHONY: install-cfg
install-cfg:
	cp etc/topbeat.template.json $(PREFIX)/topbeat.template.json
	# linux
	cp etc/topbeat.yml $(PREFIX)/topbeat-linux.yml
	# binary
	cp etc/topbeat.yml $(PREFIX)/topbeat-binary.yml
	# darwin
	cp etc/topbeat.yml $(PREFIX)/topbeat-darwin.yml
	# win
	cp etc/topbeat.yml $(PREFIX)/topbeat-win.yml

