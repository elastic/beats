#!/usr/bin/env bash

set -e

# Every fuzz test runs for 10 minutes.
# It's not recomended to run only for 10 minutes, we can add this project to oss-fuzz
# infrastructure and then run time would be 24/7.

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

# The following fuzz-test requires libpcap.
go test -fuzz=FuzzOnPacket -fuzztime=600s -run=^$ ./packetbeat/decoder
go test -fuzz=FuzzParseProcNetProto -fuzztime=600s -run=^$ ./packetbeat/procs
go test -fuzz=FuzzAmqpMessageParser -fuzztime=600s -run=^$  ./packetbeat/protos/amqp
go test -fuzz=FuzzParseDHCPv4 -fuzztime=600s -run=^$  ./packetbeat/protos/dhcpv4
go test -fuzz=FuzzParseTcp -fuzztime=600s -run=^$  ./packetbeat/protos/dns
go test -fuzz=FuzzParseUDP -fuzztime=600s -run=^$  ./packetbeat/protos/dns
go test -fuzz=FuzzDecodeDNSData -fuzztime=600s -run=^$  ./packetbeat/protos/dns
go test -fuzz=FuzzParseStream -fuzztime=600s -run=^$  ./packetbeat/protos/http
go test -fuzz=FuzzBinTryParse -fuzztime=600s -run=^$  ./packetbeat/protos/memcache
go test -fuzz=FuzzTextTryParse -fuzztime=600s -run=^$  ./packetbeat/protos/memcache
go test -fuzz=FuzzMysqlMessageParser -fuzztime=600s -run=^$ ./packetbeat/protos/mysql
go test -fuzz=FuzzParseMysqlResponse -fuzztime=600s -run=^$ ./packetbeat/protos/mysql
go test -fuzz=FuzzPgsqlMessageParser -fuzztime=600s -run=^$ ./packetbeat/protos/pgsql
go test -fuzz=FuzzParse -fuzztime=600s -run=^$ /packetbeat/protos/pgsql
go test -fuzz=FuzzParse -fuzztime=600s -run=^$ ./packetbeat/protos/redis
go test -fuzz=FuzzParseURI -fuzztime=600s -run=^$ ./packetbeat/protos/sip
go test -fuzz=FuzzParseFromToContact -fuzztime=600s -run=^$ ./packetbeat/protos/sip
go test -fuzz=ParseUDP -fuzztime=600s -run=^$ ./packetbeat/protos/sip
go test -fuzz=FuzzMessageParser -fuzztime=600s -run=^$  ./packetbeat/protos/thrift
go test -fuzz=FuzzParse -fuzztime=600s -run=^$  ./packetbeat/protos/thrift
go test -fuzz=FuzzParse -fuzztime=600s -run=^$  ./packetbeat/protos/tls

go test -fuzz=FuzzFields -fuzztime=600s -run=^$ ./x-pack/filebeat/processors/aws_vpcflow/internal/strings
go test -fuzz=FuzzUnpack -fuzztime=600s -run=^$ ./x-pack/filebeat/processors/decode_cef/cef
go test -fuzz=FuzzParse -fuzztime=600s -run=^$ ./x-pack/metricbeat/module/statsd/server 
