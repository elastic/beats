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

//go:build linux

package file_integrity

import (
	"errors"

	"github.com/elastic/elastic-agent-libs/logp"
)

func NewEventReader(c Config, logger *logp.Logger) (EventProducer, error) {
	if c.Backend == BackendAuto || c.Backend == BackendFSNotify || c.Backend == "" {
		// Auto and unset defaults to fsnotify
		l := logger.Named("fsnotify")
		l.Info("selected backend: fsnotify")
		return &fsNotifyReader{
			config:  c,
			log:     l,
			parsers: FileParsers(c),
		}, nil
	}

	if c.Backend == BackendEBPF {
		l := logger.Named("ebpf")
		l.Info("selected backend: ebpf")

		paths := make(map[string]struct{})
		for _, p := range c.Paths {
			paths[p] = struct{}{}
		}

		return &ebpfReader{
			config:  c,
			log:     l,
			parsers: FileParsers(c),
			paths:   paths,
			eventC:  make(chan Event),
		}, nil
	}

	if c.Backend == BackendKprobes {
		l := logger.Named("kprobes")
		l.Info("selected backend: kprobes")
		return &kProbesReader{
			config:  c,
			log:     l,
			parsers: FileParsers(c),
		}, nil
	}

	// unimplemented
	return nil, errors.ErrUnsupported
}
