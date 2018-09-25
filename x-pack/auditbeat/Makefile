BEAT_NAME=auditbeat
ES_BEATS?=../..
XPACK_BEAT_PATH?=github.com/elastic/beats/x-pack/${BEAT_NAME}
GOPACKAGES?=$(shell go list ${BEAT_PATH}/... ${XPACK_BEAT_PATH}/... | grep -v /vendor/ | grep -v /scripts/cmd/ )

# Include main auditbeat Makefile
include ${ES_BEATS}/${BEAT_NAME}/Makefile 
