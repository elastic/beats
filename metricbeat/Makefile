#!/bin/bash

BEATNAME=metricbeat
SYSTEM_TESTS=true
TEST_ENVIRONMENT?=true
GOPACKAGES=$(shell go list ${BEAT_DIR}/${BEATNAME}/... | grep -v /vendor/)

include ../libbeat/scripts/Makefile
