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

package log

import (
	"os"
)

// Stdin reads all incoming traffic from stdin and sends it directly to the output

func (h *Harvester) openStdin() error {
	h.source = Pipe{File: os.Stdin}

	var err error
	h.encoding, err = h.encodingFactory(h.source)

	return err
}

// restrict file to minimal interface of FileSource to prevent possible casts
// to additional interfaces supported by underlying file
type Pipe struct {
	File *os.File
}

func (p Pipe) Read(b []byte) (int, error) { return p.File.Read(b) }
func (p Pipe) Close() error               { return p.File.Close() }
func (p Pipe) Name() string               { return p.File.Name() }
func (p Pipe) Stat() (os.FileInfo, error) { return p.File.Stat() }
func (p Pipe) Continuable() bool          { return false }
func (p Pipe) HasState() bool             { return false }
func (p Pipe) Removed() bool              { return false }
