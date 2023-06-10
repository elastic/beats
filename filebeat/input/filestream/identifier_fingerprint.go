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

package filestream

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	DefaultFingerprintSize int64 = 1024 // 1KB
)

type fingerprintConfig struct {
	Offset int64 `config:"offset"`
	Length int64 `config:"length"`
}

type fingerprintIdentifier struct {
	log *logp.Logger
	cfg fingerprintConfig
}

func newFingerprintIdentifier(cfg *conf.C) (fileIdentifier, error) {
	config := fingerprintConfig{
		Length: DefaultFingerprintSize,
	}

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("error while reading configuration of fingerprint file identity: %w", err)
	}
	if config.Length < sha256.BlockSize {
		err := fmt.Errorf("fingerprint size %d cannot be smaller than %d", config.Length, sha256.BlockSize)
		return nil, fmt.Errorf("error while reading configuration of fingerprint file identity: %w", err)
	}

	return &fingerprintIdentifier{
		log: logp.NewLogger("fingerprint_identifier"),
		cfg: config,
	}, nil
}

func (i *fingerprintIdentifier) getFingerprint(filename string) (fingerprint string, err error) {
	h := sha256.New()
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("failed to open %q for fingerprinting: %w", filename, err)
	}
	defer file.Close()

	if i.cfg.Offset != 0 {
		_, err = file.Seek(i.cfg.Offset, io.SeekStart)
		if err != nil {
			return "", fmt.Errorf("failed to seek %q for fingerprinting: %w", filename, err)
		}
	}

	r := io.LimitReader(file, i.cfg.Length)
	buf := make([]byte, h.BlockSize())
	_, err = io.CopyBuffer(h, r, buf)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash for first %d bytes of %q: %w", i.cfg.Length, filename, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (i *fingerprintIdentifier) GetSource(e loginp.FSEvent) (src fileSource, err error) {
	fileSize := e.Info.Size()
	if fileSize < i.cfg.Offset+i.cfg.Length {
		return src, fmt.Errorf("filesize - %d, expected at least - %d: %w", fileSize, i.cfg.Offset+i.cfg.Length, ErrFileSizeTooSmall)
	}

	var filename string
	if e.Op == loginp.OpDelete {
		filename = e.OldPath
	} else {
		filename = e.NewPath
	}
	fingerprint, err := i.getFingerprint(filename)
	return fileSource{
		info:                e.Info,
		newPath:             e.NewPath,
		oldPath:             e.OldPath,
		truncated:           e.Op == loginp.OpTruncate,
		archived:            e.Op == loginp.OpArchived,
		fileID:              fingerprintName + identitySep + fingerprint,
		identifierGenerator: fingerprintName,
	}, err
}

func (i *fingerprintIdentifier) Name() string {
	return fingerprintName
}

func (i *fingerprintIdentifier) Supports(f identifierFeature) bool {
	switch f {
	case trackRename:
		return true
	default:
	}
	return false
}
