#!/bin/sh

set -e


BEAT_PATH=/go/src/github.com/elastic/winlogbeat
cd $BEAT_PATH
PREFIX=/build make deps install-cfg
