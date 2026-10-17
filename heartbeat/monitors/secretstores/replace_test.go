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

package secretstores

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	ucfg "github.com/elastic/go-ucfg"
)

func TestSchemeReplacer(t *testing.T) {
	t.Setenv("SECRETSTORES_TEST_ENV", "from-env")
	t.Setenv("SECRETSTORES_TEST_REF", "demo://safe/item#user")

	r := NewSchemeReplacer("demo")
	check := func(content string) error {
		if !strings.Contains(content, "#") {
			return errors.New("missing #field")
		}
		return nil
	}
	resolve := func(content string) (string, error) {
		if err := check(content); err != nil {
			return "", err
		}
		return "<" + content + ">", nil
	}

	ntlmBase64 := base64.StdEncoding.EncodeToString([]byte(`{"domain": "${SECRETSTORES_TEST_ENV}", "password": "demo://safe/item#pw"}`))
	script := "step('login', () => page.fill('#pw', 'demo://bad'))"

	raw := map[string]any{
		"header": "Bearer demo://safe/item#tok",
		"env":    "${SECRETSTORES_TEST_REF}",
		"ntlm":   ntlmBase64,
		"source": map[string]any{"inline": map[string]any{"script": script}},
	}
	u, err := ucfg.NewFrom(raw, ucfg.PathSep(""), ucfg.VarExp)
	require.NoError(t, err, "test config must be valid")
	cfg := (*conf.C)(u)

	require.NoError(t, r.Validate(cfg, check), "valid references and skipped inline script must pass validation")

	got, err := r.Replace(cfg, resolve)
	require.NoError(t, err, "references must resolve")
	var resolved map[string]any
	require.NoError(t, got.Unpack(&resolved), "resolved config must unpack")
	assert.Equal(t, map[string]any{
		"header": "Bearer <safe/item#tok>",
		"env":    "<safe/item#user>",
		"ntlm":   map[string]any{"domain": "from-env", "password": "<safe/item#pw>"},
		"source": map[string]any{"inline": map[string]any{"script": script}},
	}, resolved, "unexpected resolved config")
}
