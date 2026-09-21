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
Package auditd provides a filestream parser for Linux audit log lines.

The parser operates in one of three modes, controlled by the "mode" config
setting:

  - parse (default): each input line is parsed as an independent audit record, populating
    auditd.log.* fields (record type, sequence number, and all key=value pairs)
    and setting the message timestamp from the audit header. Architecture codes
    (e.g. c000003e → x86_64), syscall numbers (e.g. 59 → execve), and
    res=success normalisation are resolved at parse time.
  - coalesce: records sharing the same audit sequence number are grouped using
    go-libaudit's Reassembler and coalesced into compound events via
    aucoalesce.CoalesceMessages. The output uses the auditd.data.* namespace
    (matching the auditd_manager integration) and populates ECS root fields
    (process.*, user.*, event.*, file.*).
  - none: pass-through, no parsing.

Lines written by userspace auditd with name_format=hostname in
/etc/audit/auditd.conf carry a "node=<hostname> " prefix that the kernel
does not emit. Both modes strip this prefix and expose the hostname (as
auditd.log.node in parse mode, auditd.data.node in coalesce mode).

The parser is registered under the name "auditd" in
libbeat/reader/parser/parser.go and is configurable through Config:

  - mode          – parser mode: none, parse, or coalesce (default: parse)
  - log_errors    – log parse errors via the logger (default: false)
  - add_error_key – add an error.message field to the event on parse
    failure (default: true)

Note: the implementation is Linux-only because go-libaudit's auparse
package depends on linux/unix signal name lookups. A build stub is provided
for other platforms so the package compiles cross-platform.
*/
package auditd
