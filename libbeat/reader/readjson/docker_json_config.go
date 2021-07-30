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

package readjson

import "fmt"

type ContainerFormat uint8

type Stream uint8

const (
	Auto ContainerFormat = iota + 1
	CRI
	Docker
	JSONFile

	All Stream = iota + 1
	Stdout
	Stderr
)

var (
	containerFormats = map[string]ContainerFormat{
		"auto":      Auto,
		"cri":       CRI,
		"docker":    Docker,
		"json-file": JSONFile,
	}

	containerStreams = map[string]Stream{
		"all":    All,
		"stdout": Stdout,
		"stderr": Stderr,
	}
)

type ContainerJSONConfig struct {
	Stream Stream          `config:"stream"`
	Format ContainerFormat `config:"format"`
}

func DefaultContainerConfig() ContainerJSONConfig {
	return ContainerJSONConfig{
		Format: Auto,
		Stream: All,
	}
}

func (f *ContainerFormat) Unpack(v string) error {
	val, ok := containerFormats[v]
	if !ok {
		keys := make([]string, len(containerFormats))
		i := 0
		for k := range containerFormats {
			keys[i] = k
			i++
		}
		return fmt.Errorf("unknown container log format: %s, supported values: %+v", v, keys)
	}
	*f = val
	return nil
}

func (s *Stream) Unpack(v string) error {
	val, ok := containerStreams[v]
	if !ok {
		keys := make([]string, len(containerStreams))
		i := 0
		for k := range containerStreams {
			keys[i] = k
			i++
		}
		return fmt.Errorf("unknown streams: %s, supported values: %+v", v, keys)
	}
	*s = val
	return nil
}

func (s *Stream) String() string {
	for k, v := range containerStreams {
		if v == *s {
			return k
		}
	}
	return ""
}
