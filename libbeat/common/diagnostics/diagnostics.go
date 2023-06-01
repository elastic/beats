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

package diagnostics

// DiagnosticReporter is an interface that a metricset, fileset, or runner should implement to provide additional Diagnostic data.
// A DiagnosticReporter can provide any number of diagnostic responses when requested.
type DiagnosticReporter interface {
	// Diagnostics returns metadata and a callback handler.
	// note that this can be called any time after a metricset has started, so implementors should not assume
	// the state of a metricset/fileset when this method is called.
	Diagnostics() []DiagnosticSetup
}

// DiagnosticSetup contains the data needed to register a callback.
type DiagnosticSetup struct {
	// The name of this diagnostics data result.
	Name string
	// A brief description of the file.
	Description string
	// The filename that the requester should save the body as. This value must be unique for all other diagnostics in the metricset/fileset
	Filename string
	// MIME/ContentType. See https://www.iana.org/assignments/media-types/media-types.xhtml
	ContentType string
	//Callback is called when diagnostic data is actually requested by central management.
	// Callback does not return an error, and if one occours, it should be written out as the result.
	Callback func() []byte
}
