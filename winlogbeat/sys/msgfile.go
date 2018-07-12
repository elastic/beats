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

package sys

// MessageFiles contains handles to event message files associated with an
// event log source.
type MessageFiles struct {
	SourceName string
	Err        error
	Handles    []FileHandle
}

// FileHandle contains the handle to a single Windows message file.
type FileHandle struct {
	File   string  // Fully-qualified path to the event message file.
	Handle uintptr // Handle to the loaded event message file.
	Err    error   // Error that occurred while loading Handle.
}
