BEAT_NAME=auditbeat
ES_BEATS=../..
XPACK_BEAT_PATH?=github.com/elastic/beats/x-pack/${BEAT_NAME}
GOPACKAGES=$(shell go list ${BEAT_PATH}/... ${XPACK_BEAT_PATH}/... | grep -v /vendor/ | grep -v /scripts/cmd/ )
FIND=. ${PYTHON_ENV}/bin/activate; find . ${ES_BEATS}/${BEAT_NAME} -type f -not -path "*/vendor/*" -not -path "*/build/*" -not -path "*/.git/*"

# Include main auditbeat Makefile
include ${ES_BEATS}/${BEAT_NAME}/Makefile

# Overwrite check-headers - check for Apache license in
# auditbeat/ and Elastic license in xpack/auditbeat/
.PHONY: check-headers
check-headers:
ifndef CHECK_HEADERS_DISABLED
	@go get -u github.com/elastic/go-licenser
	@go-licenser -d -license ${LICENSE} ${ES_BEATS}/${BEAT_NAME}
	@go-licenser -d -license Elastic .
endif

# Overwrite check-headers - insert Apache license in
# auditbeat/ and Elastic license in xpack/auditbeat/
.PHONY: add-headers
add-headers:
ifndef CHECK_HEADERS_DISABLED
	@go get github.com/elastic/go-licenser
	@go-licenser -license ${LICENSE} ${ES_BEATS}/${BEAT_NAME}
	@go-licenser -license Elastic .
endif