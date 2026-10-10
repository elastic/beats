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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	conf "github.com/elastic/elastic-agent-libs/config"
	ucfg "github.com/elastic/go-ucfg"
)

// entry is one validated secret_stores item.
type entry struct {
	storeType string
	config    *conf.C
	// identity is a hash of the whole entry, credentials included. It never
	// leaves the process and is not logged.
	identity string
}

// parseEntries converts the unpacked secret_stores value into entries.
//
// The value is either a list of objects or a base64 string of a YAML/JSON
// list, the form used by Fleet. Base64 content is not variable-expanded, so
// credentials containing "${" are kept as they are.
func parseEntries(raw any) ([]entry, error) {
	var items []any
	switch v := raw.(type) {
	case nil:
		return nil, nil
	case string:
		decoded, err := decodeEntries(v)
		if err != nil {
			return nil, err
		}
		items = decoded
	case []any:
		items = v
	default:
		return nil, fmt.Errorf("%s: expected a list or a base64 string, got %T", ConfigKey, raw)
	}

	entries := make([]entry, 0, len(items))
	seen := map[string]bool{}
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: entry %d: expected an object, got %T", ConfigKey, i, item)
		}
		storeType, _ := m["type"].(string)
		if storeType == "" {
			return nil, fmt.Errorf("%s: entry %d: 'type' is required", ConfigKey, i)
		}
		if _, ok := lookupFactory(storeType); !ok {
			return nil, fmt.Errorf("%s: entry %d: unknown secret store type %q", ConfigKey, i, storeType)
		}
		if seen[storeType] {
			return nil, fmt.Errorf("%s: only one %q entry is supported", ConfigKey, storeType)
		}
		seen[storeType] = true

		// PathSep("") keeps keys literal; values are already resolved and
		// must not be expanded again.
		cfg, err := ucfg.NewFrom(m, ucfg.PathSep(""))
		if err != nil {
			return nil, fmt.Errorf("%s: entry %d: %w", ConfigKey, i, err)
		}
		identity, err := entryIdentity(m)
		if err != nil {
			return nil, fmt.Errorf("%s: entry %d: %w", ConfigKey, i, err)
		}
		entries = append(entries, entry{storeType: storeType, config: (*conf.C)(cfg), identity: identity})
	}
	return entries, nil
}

func decodeEntries(s string) ([]any, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("%s: invalid base64: %w", ConfigKey, err)
	}
	var items []any
	if err := yaml.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("%s: decoded value is not a YAML/JSON list: %w", ConfigKey, err)
	}
	return items, nil
}

func entryIdentity(m map[string]any) (string, error) {
	// encoding/json sorts map keys, so equal entries have equal identities.
	b, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("cannot compute identity: %w", err)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
