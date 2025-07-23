#!/usr/bin/env bash

set -e

# Every fuzz test runs for 10 minutes.
# It's not recomended to run only for 10 minutes, we can add this project to oss-fuzz
# infrastructure and then run time would be 24/7.

go test -fuzz=FuzzIsRFC5424Format -fuzztime=600s -run=^$ ./filebeat/input/syslog
go test -fuzz=FuzzParserRFC3164 -fuzztime=600s -run=^$ ./filebeat/input/syslog
go test -fuzz=FuzzParserRFC5424 -fuzztime=600s -run=^$ ./filebeat/input/syslog
