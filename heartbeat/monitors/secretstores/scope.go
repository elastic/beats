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
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// Scope tells a Store which of the references of its type are its own, when
// several connections of the same type are configured. References carry an
// optional connection name: <scheme>://<name>@<content>.
//
// A reference with a name belongs to the entry with that name. A reference
// without a name belongs to the default entry: the only entry of its type, or
// else the only one without a name.
type Scope struct {
	// Name is the name of the entry, empty if it has none.
	Name string
	// Default is true if the references without a name belong to the entry.
	Default bool
	// HasDefault is true if one of the entries of the type is the default.
	HasDefault bool
	// Names are the names of all the entries of the type.
	Names []string
}

// SplitName splits the content of a reference into its connection name and
// the rest. The name is the text before an "@" that comes before any "/" or
// "#", and is percent-encoded. hasName is false for a reference without name.
func SplitName(content string) (name, rest string, hasName bool) {
	end := strings.IndexAny(content, "@/#")
	if end < 0 || content[end] != '@' {
		return "", content, false
	}
	return content[:end], content[end+1:], true
}

// SelectionError is returned by Claim when no entry can resolve a reference.
// The fault is in the reference or in the list of entries, not in the entry
// that reported it.
type SelectionError struct{ error }

func (e *SelectionError) Unwrap() error { return e.error }

// Claim reports whether the reference with the given content belongs to the
// scope, and returns the content without the connection name. It fails for
// references that no entry can resolve.
func (s Scope) Claim(content string) (rest string, mine bool, err error) {
	rawName, rest, hasName := SplitName(content)
	if !hasName {
		switch {
		case s.Default:
			return rest, true, nil
		case s.HasDefault:
			return "", false, nil
		}
		return "", false, &SelectionError{fmt.Errorf("the reference has no connection name, but there are several entries: use <scheme>://<name>@... with one of %s", quoteAll(s.Names))}
	}

	name, err := url.PathUnescape(rawName)
	if err != nil {
		return "", false, fmt.Errorf("invalid connection name: %w", err)
	}
	switch {
	case name == "":
		return "", false, &SelectionError{errors.New("empty connection name")}
	case name == s.Name:
		return rest, true, nil
	case slices.Contains(s.Names, name):
		return "", false, nil
	}
	if len(s.Names) == 0 {
		return "", false, &SelectionError{fmt.Errorf("no entry has the name %q, set 'name: %s' on the entry to use it", name, name)}
	}
	return "", false, &SelectionError{fmt.Errorf("no entry has the name %q, available names: %s", name, quoteAll(s.Names))}
}

func quoteAll(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = fmt.Sprintf("%q", n)
	}
	return strings.Join(q, ", ")
}

// scopes returns the Scope of each entry, in the order of entries.
func scopes(entries []entry) []Scope {
	counts := map[string]int{}
	names := map[string][]string{}
	hasDefault := map[string]bool{}
	for _, e := range entries {
		counts[e.storeType]++
		if e.name != "" {
			names[e.storeType] = append(names[e.storeType], e.name)
		}
	}
	for _, e := range entries {
		hasDefault[e.storeType] = hasDefault[e.storeType] || e.name == "" || counts[e.storeType] == 1
	}

	out := make([]Scope, len(entries))
	for i, e := range entries {
		slices.Sort(names[e.storeType])
		out[i] = Scope{
			Name:       e.name,
			Default:    e.name == "" || counts[e.storeType] == 1,
			HasDefault: hasDefault[e.storeType],
			Names:      names[e.storeType],
		}
	}
	return out
}
