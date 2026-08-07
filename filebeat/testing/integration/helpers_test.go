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

//go:build integration

package integration

import (
	libbeatintegration "github.com/elastic/beats/v7/libbeat/testing/integration"
)

// networkReportOptions is the shared ReportOptions for all network input tests.
var networkReportOptions = libbeatintegration.ReportOptions{
	PrintLinesOnFail:  100,
	PrintConfigOnFail: true,
}

// exitSignalKilled is the exit code returned by Go's exec package when a
// process is terminated by a signal (SIGKILL on Unix). Used with ExpectStop
// in tests that cancel the context to stop the beat rather than relying on
// a watcher-triggered kill.
const exitSignalKilled = -1

// Log substrings emitted by the filebeat network inputs when ready.
// Centralised here to avoid silent breakage if the source messages change.
const (
	tcpReadyMsg = "Start accepting connections"
	udpReadyMsg = "Started listening for UDP connection"
)
