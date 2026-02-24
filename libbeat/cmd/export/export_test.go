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

package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/elastic/beats/v7/libbeat/version"
	libversion "github.com/elastic/elastic-agent-libs/version"
	"github.com/stretchr/testify/require"
)

func TestNewStdoutClientInvalidVersion(t *testing.T) {
	client, err := newStdoutClient("not-a-version")
	require.Error(t, err)
	require.Nil(t, client)
}

func TestNewStdoutClientDefaultVersion(t *testing.T) {
	client, err := newStdoutClient("")
	require.NoError(t, err)
	require.Equal(t, *libversion.MustNew(version.GetDefaultVersion()), client.GetVersion())
}

func TestFileClientWrite(t *testing.T) {
	client, err := newFileClient(t.TempDir(), "8.17.0")
	require.NoError(t, err)

	err = client.Write("template", "my-index", `{"index":"ok"}`)
	require.NoError(t, err)

	path := filepath.Join(client.dir, "template", "my-index.json")
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, `{"index":"ok"}`, string(body))
}

func TestFileClientWriteReturnsErrorWhenComponentPathIsFile(t *testing.T) {
	base := t.TempDir()
	componentPath := filepath.Join(base, "template")
	require.NoError(t, os.WriteFile(componentPath, []byte("not-a-dir"), 0o644))

	client := &fileClient{dir: base}
	err := client.Write("template", "my-index", `{"index":"ok"}`)
	require.Error(t, err)
}
