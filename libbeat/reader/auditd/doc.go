// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

/*
Package auditd provides a filestream parser that pre-parses Linux audit log
lines on the agent using go-libaudit's auparse package before events are
shipped to Elasticsearch.

Each line is processed by auparse.ParseLogLine, which populates
auditd.log.* fields (record type, sequence number, and all key=value pairs)
and sets the message timestamp from the audit header. Architecture codes
(e.g. c000003e → x86_64), syscall numbers (e.g. 59 → execve), and
res=success normalisation are resolved by auparse at parse time.

Lines written by userspace auditd with name_format=hostname in
/etc/audit/auditd.conf carry a "node=<hostname> " prefix that the kernel
does not emit. The parser strips this prefix and exposes the hostname as
the auditd.log.node field.

The parser is registered under the name "auditd" in
libbeat/reader/parser/parser.go and is configurable through Config:

  - log_errors   – log parse errors via the logger (default: false)
  - add_error_key – add an error.message field to the event on parse
    failure (default: true)

Compatibility: this parser replaces the grok/Painless ingest pipeline used
by the auditd integration (elastic/integrations). The integration provides
a use_filebeat_parser toggle (default false) so users can opt in and
validate before the pipeline path is retired.

Note: the implementation is Linux-only because go-libaudit's auparse
package depends on linux/unix signal name lookups. A build stub is provided
for other platforms so the package compiles cross-platform.
*/
package auditd
