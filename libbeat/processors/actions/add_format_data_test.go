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

package actions

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/formats/elf"
	"github.com/elastic/beats/v7/libbeat/formats/lnk"
	"github.com/elastic/beats/v7/libbeat/formats/macho"
	"github.com/elastic/beats/v7/libbeat/formats/pe"
)

func TestFormatDataPE(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"foo.bar.baz": "../../formats/fixtures/pe/hello-windows",
		},
	}
	p, err := NewAddFormatData(common.MustNewConfigFrom(map[string]interface{}{
		"field": "foo.bar.baz",
	}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	data, err := observed.Fields.GetValue("file.pe")
	require.NoError(t, err)
	_, ok := data.(*pe.Info)
	require.True(t, ok)
}

func TestFormatDataMachO(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"foo.bar.baz": "../../formats/fixtures/macho/hello-darwin",
		},
	}
	p, err := NewAddFormatData(common.MustNewConfigFrom(map[string]interface{}{
		"field": "foo.bar.baz",
	}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	data, err := observed.Fields.GetValue("file.macho")
	require.NoError(t, err)
	_, ok := data.(*macho.Info)
	require.True(t, ok)
}

func TestFormatDataElf(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"foo.bar.baz": "../../formats/fixtures/elf/hello-linux",
		},
	}
	p, err := NewAddFormatData(common.MustNewConfigFrom(map[string]interface{}{
		"field": "foo.bar.baz",
	}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	data, err := observed.Fields.GetValue("file.elf")
	require.NoError(t, err)
	_, ok := data.(*elf.Info)
	require.True(t, ok)
}

func TestFormatDataLnk(t *testing.T) {
	evt := beat.Event{
		Fields: common.MapStr{
			"foo.bar.baz": "../../formats/fixtures/lnk/local_cmd.lnk",
		},
	}
	p, err := NewAddFormatData(common.MustNewConfigFrom(map[string]interface{}{
		"field": "foo.bar.baz",
	}))
	require.NoError(t, err)
	observed, err := p.Run(&evt)
	require.NoError(t, err)
	data, err := observed.Fields.GetValue("file.lnk")
	require.NoError(t, err)
	_, ok := data.(*lnk.Info)
	require.True(t, ok)
}
