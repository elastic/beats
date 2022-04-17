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

package path_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/processors/script/javascript"

	_ "github.com/menderesk/beats/v7/libbeat/processors/script/javascript/module/path"
	_ "github.com/menderesk/beats/v7/libbeat/processors/script/javascript/module/require"
)

func TestWin32(t *testing.T) {
	const script = `
var path = require('path');

function process(evt) {
    var filename = "C:\\Windows\\system32\\..\\system32\\system32.dll";
	evt.Put("result", {
        raw: filename,
    	basename: path.win32.basename(filename),
    	dirname:  path.win32.dirname(filename),
    	extname: path.win32.extname(filename),
    	isAbsolute: path.win32.isAbsolute(filename),
    	normalize: path.win32.normalize(filename),
        sep: path.win32.sep,
    });
}
`

	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(&beat.Event{Fields: common.MapStr{}})
	if err != nil {
		t.Fatal(err)
	}

	fields := evt.Fields.Flatten()
	assert.Equal(t, "system32.dll", fields["result.basename"])
	assert.Equal(t, `C:\Windows\system32`, fields["result.dirname"])
	assert.Equal(t, ".dll", fields["result.extname"])
	assert.Equal(t, true, fields["result.isAbsolute"])
	assert.Equal(t, `C:\Windows\system32\system32.dll`, fields["result.normalize"])
	assert.EqualValues(t, '\\', fields["result.sep"])
}

func TestPosix(t *testing.T) {
	const script = `
var path = require('path');

function process(evt) {
    var filename = "/usr/lib/../lib/libcurl.so";
	evt.Put("result", {
        raw: filename,
    	basename: path.posix.basename(filename),
    	dirname:  path.posix.dirname(filename),
    	extname: path.posix.extname(filename),
    	isAbsolute: path.posix.isAbsolute(filename),
    	normalize: path.posix.normalize(filename),
        sep: path.posix.sep,
    });
}
`

	p, err := javascript.NewFromConfig(javascript.Config{Source: script}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(&beat.Event{Fields: common.MapStr{}})
	if err != nil {
		t.Fatal(err)
	}

	fields := evt.Fields.Flatten()
	assert.Equal(t, "libcurl.so", fields["result.basename"])
	assert.Equal(t, "/usr/lib", fields["result.dirname"])
	assert.Equal(t, ".so", fields["result.extname"])
	assert.Equal(t, true, fields["result.isAbsolute"])
	assert.Equal(t, "/usr/lib/libcurl.so", fields["result.normalize"])
	assert.EqualValues(t, '/', fields["result.sep"])
}
