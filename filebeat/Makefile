#!/bin/bash

BEATNAME=filebeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT=false

include ../libbeat/scripts/Makefile

# This is called by the beats packer before building starts
.PHONY: before-build
before-build:
	# linux
	sed -i 's@#registry_file: .filebeat@registry_file: /var/lib/filebeat/registry@' $(PREFIX)/filebeat-linux.yml
	# win
	sed -i 's@#registry_file: .filebeat@registry_file: "C:/ProgramData/filebeat/registry"@' $(PREFIX)/filebeat-win.yml
