#!/usr/bin/env bash

set -e

# Every fuzz test runs for 10 minutes.
# It's not recomended to run only for 10 minutes, we can add this project to oss-fuzz
# infrastructure and then run time would be 24/7.

go test -fuzz=FuzzIsRFC5424Format -fuzztime=600s -run=^$ ./filebeat/input/syslog
go test -fuzz=FuzzParserRFC3164 -fuzztime=600s -run=^$ ./filebeat/input/syslog
go test -fuzz=FuzzParserRFC5424 -fuzztime=600s -run=^$ ./filebeat/input/syslog

go test -fuzz=FuzzFormat -fuzztime=600s -run=^$ ./libbeat/common/dtfmt
go test -fuzz=FuzzNew -fuzztime=600s -run=^$ ./libbeat/processors/dissect
go test -fuzz=FuzzParseRFC3164 -fuzztime=600s -run=^$ ./libbeat/reader/syslog
go test -fuzz=FuzzParseRFC5424 -fuzztime=600s -run=^$ ./libbeat/reader/syslog
go test -fuzz=FuzzIsRFC5424 -fuzztime=600s -run=^$ ./libbeat/reader/syslog
go test -fuzz=FuzzParseStructuredData -fuzztime=600s -run=^$ ./libbeat/reader/syslog

go test -fuzz=FuzzParseMetricFamilies -fuzztime=600s -run=^$ ./metricbeat/helper/prometheus
go test -fuzz=FuzzSplitTagsFromMetricName -fuzztime=600s -run=^$ ./metricbeat/module/dropwizard/collector
go test -fuzz=FuzzProcess -fuzztime=600s -run=^$ ./metricbeat/module/graphite/server
go test -fuzz=FuzzParseMBeanName -fuzztime=600s -run=^$ ./metricbeat/module/jolokia/jmx
go test -fuzz=FuzzParseSrvr -fuzztime=600s -run=^$ ./metricbeat/module/zookeeper/server
