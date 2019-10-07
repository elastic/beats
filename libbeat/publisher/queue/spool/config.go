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

package spool

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/joeshaw/multierror"

	"github.com/elastic/beats/libbeat/common/cfgtype"
)

type config struct {
	File  pathConfig  `config:"file"`
	Write writeConfig `config:"write"`
	Read  readConfig  `config:"read"`
}

type pathConfig struct {
	Path        string           `config:"path"`
	Permissions os.FileMode      `config:"permissions"`
	MaxSize     cfgtype.ByteSize `config:"size"`
	PageSize    cfgtype.ByteSize `config:"page_size"`
	Prealloc    bool             `config:"prealloc"`
}

type writeConfig struct {
	BufferSize   cfgtype.ByteSize `config:"buffer_size"`
	FlushEvents  int              `config:"flush.events"`
	FlushTimeout time.Duration    `config:"flush.timeout"`
	Codec        codecID          `config:"codec"`
}

type readConfig struct {
	FlushTimeout time.Duration `config:"flush.timeout"`
}

func defaultConfig() config {
	return config{
		File: pathConfig{
			Path:        "",
			Permissions: 0600,
			MaxSize:     100 * humanize.MiByte,
			PageSize:    4 * humanize.KiByte,
			Prealloc:    true,
		},
		Write: writeConfig{
			BufferSize:   1 * humanize.MiByte,
			FlushTimeout: 1 * time.Second,
			FlushEvents:  16 * 1024,
			Codec:        codecCBORL,
		},
		Read: readConfig{
			FlushTimeout: 0,
		},
	}
}

func (c *pathConfig) Validate() error {
	var errs multierror.Errors

	if c.MaxSize < humanize.MiByte {
		errs = append(errs, errors.New("max size must be larger 1MiB"))
	}

	if !c.Permissions.IsRegular() {
		errs = append(errs, fmt.Errorf("permissions %v are not regular file permissions", c.Permissions.String()))
	} else {
		m := c.Permissions.Perm()
		if (m & 0400) == 0 {
			errs = append(errs, errors.New("file must be readable by current user"))
		}
		if (m & 0200) == 0 {
			errs = append(errs, errors.New("file must be writable by current user"))
		}
	}

	// TODO: good 'limit' on pageSize?

	if c.PageSize >= c.MaxSize {
		errs = append(errs, fmt.Errorf("page_size (%v) must be less then size (%v)", c.PageSize, c.MaxSize))
	}

	return errs.Err()
}

func (c *writeConfig) Validate() error {
	return nil
}

func (c *readConfig) Validate() error {
	return nil
}

func (c *codecID) Unpack(value string) error {
	ids := map[string]codecID{
		"json":   codecJSON,
		"ubjson": codecUBJSON,
		"cbor":   codecCBORL,
	}

	id, exists := ids[strings.ToLower(value)]
	if !exists {
		return fmt.Errorf("codec '%v' not available", value)
	}

	*c = id
	return nil
}
