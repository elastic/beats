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

package mage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreatePacker scaffolds a dev-tools/packer directory for a beat.
// Replaces libbeat/scripts/create_packer.py.
func CreatePacker(beat, beatPath, esBeats, version string) error {
	absPath, _ := filepath.Abs(".")
	esBeatsAbs, _ := filepath.Abs(esBeats)

	packerPath := filepath.Join(absPath, "dev-tools", "packer")
	fmt.Println(packerPath)

	if info, err := os.Stat(packerPath); err == nil && info.IsDir() {
		fmt.Println("Dev tools already exists. Stopping...")
		return nil
	}

	if err := os.MkdirAll(filepath.Join(packerPath, "beats"), 0o755); err != nil {
		return fmt.Errorf("creating directories: %w", err)
	}

	templates := filepath.Join(esBeatsAbs, "libbeat", "scripts", "dev-tools", "packer")

	if err := writePackerTemplate(filepath.Join(templates, "version.yml"), filepath.Join(packerPath, "version.yml"), beat, beatPath, version); err != nil {
		return err
	}
	if err := writePackerTemplate(filepath.Join(templates, "Makefile"), filepath.Join(packerPath, "Makefile"), beat, beatPath, version); err != nil {
		return err
	}
	if err := writePackerTemplate(filepath.Join(templates, "config.yml"), filepath.Join(packerPath, "beats", beat+".yml"), beat, beatPath, version); err != nil {
		return err
	}

	fmt.Println("Packer directories created")
	return nil
}

func writePackerTemplate(src, dst, beat, beatPath, version string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", src, err)
	}

	content := strings.ReplaceAll(string(data), "{beat}", beat)
	content = strings.ReplaceAll(content, "{beat_path}", beatPath)
	content = strings.ReplaceAll(content, "{version}", version)

	return os.WriteFile(dst, []byte(content), 0o644)
}
