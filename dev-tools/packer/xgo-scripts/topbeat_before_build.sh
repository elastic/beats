#!/bin/sh

set -e

BEAT_PATH=/go/src/github.com/elastic/beats/topbeat
cd $BEAT_PATH
PREFIX=/build make install-cfg
