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

package http

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// kerberosSettings wraps kerberos.Config so the `kerberos` field can unpack
// either a nested object (heartbeat.yml) or a base64-encoded YAML/JSON blob
// (synthetics integration / Fleet).
type kerberosSettings struct {
	*kerberos.Config
}

func (k *kerberosSettings) IsEnabled() bool {
	return k != nil && k.Config.IsEnabled()
}

func (k *kerberosSettings) Unpack(v any) error {
	if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
		k.Config = nil
		return nil
	}
	inner := &kerberos.Config{}
	if err := unpackYAMLOrBase64(v, inner, "kerberos"); err != nil {
		return err
	}
	k.Config = inner
	return nil
}

// ntlmPlain is NTLMConfig without Unpack, so ucfg can fill the struct tags
// without recursing into NTLMConfig.Unpack.
type ntlmPlain NTLMConfig

func (n *NTLMConfig) Unpack(v any) error {
	if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
		disabled := false
		n.Enabled = &disabled
		return nil
	}
	var plain ntlmPlain
	if err := unpackYAMLOrBase64(v, &plain, "ntlm"); err != nil {
		return err
	}
	*n = NTLMConfig(plain)
	return nil
}

func unpackYAMLOrBase64(v any, dest any, name string) error {
	switch val := v.(type) {
	case string:
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(val))
		if err != nil {
			return fmt.Errorf("%s: invalid base64: %w", name, err)
		}
		cfg, err := conf.NewConfigWithYAML(raw, name)
		if err != nil {
			return fmt.Errorf("%s: invalid yaml: %w", name, err)
		}
		return cfg.Unpack(dest)
	case map[string]any:
		cfg, err := conf.NewConfigFrom(val)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return cfg.Unpack(dest)
	default:
		return fmt.Errorf("%s: expected object or base64 string, got %T", name, v)
	}
}
