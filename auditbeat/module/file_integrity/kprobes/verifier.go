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

package kprobes

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"strings"

	tkbtf "github.com/elastic/tk-btf"
)

//go:embed embed
var embedBTFFolder embed.FS

func loadAllSpecs() ([]*tkbtf.Spec, error) {
	var specs []*tkbtf.Spec

	spec, err := tkbtf.NewSpecFromKernel()
	if err != nil {
		if !errors.Is(err, tkbtf.ErrSpecKernelNotSupported) {
			return nil, err
		}
	} else {
		specs = append(specs, spec)
	}

	embeddedSpecs, err := loadEmbeddedSpecs()
	if err != nil {
		return nil, err
	}
	specs = append(specs, embeddedSpecs...)
	return specs, nil
}

func loadEmbeddedSpecs() ([]*tkbtf.Spec, error) {
	var specs []*tkbtf.Spec
	err := fs.WalkDir(embedBTFFolder, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !strings.HasSuffix(path, ".btf") {
			return nil
		}

		embedFileBytes, err := embedBTFFolder.ReadFile(path)
		if err != nil {
			return err
		}

		embedSpec, err := tkbtf.NewSpecFromReader(bytes.NewReader(embedFileBytes), nil)
		if err != nil {
			return err
		}

		specs = append(specs, embedSpec)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return specs, nil
}
