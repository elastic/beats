#
# Includes
#
include ../dev-tools/make/oss.mk

#
# Targets
#

.PHONY: benchmark
benchmark:
	go test -short -bench=. ./... -cpu=2

.PHONY: create-tcp-protocol
create-tcp-protocol: python-env
	@${PYTHON_VE_DIR}/bin/python ${ES_BEATS}/packetbeat/scripts/create_tcp_protocol.py
