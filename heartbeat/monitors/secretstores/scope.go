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
// several connections of the same type are configured. Every entry has a
// name, and every reference carries it: <scheme>://<name>@<content>.
type Scope struct {
	// Name is the name of the entry.
	Name string
	// AllNames are the names of all the entries of the type, Name included.
	AllNames []string
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

// ErrSelection is wrapped by the errors of Claim for references that no entry
// can resolve. The fault is in the reference or in the list of entries, not in
// the entry that reported it.
var ErrSelection = errors.New("invalid connection selection")

// Claim reports whether the reference with the given content belongs to the
// scope, and returns the content without the connection name. It fails for
// references that no entry can resolve.
func (s Scope) Claim(content string) (rest string, mine bool, err error) {
	rawName, rest, hasName := SplitName(content)
	if !hasName {
		return "", false, fmt.Errorf("%w: the reference must name its connection, as <scheme>://<name>@..., one of %s", ErrSelection, quoteAll(s.AllNames))
	}

	name, err := url.PathUnescape(rawName)
	if err != nil {
		return "", false, errors.New("invalid percent-encoding in connection name")
	}
	switch {
	case name == "":
		return "", false, fmt.Errorf("%w: empty connection name", ErrSelection)
	case name == s.Name:
		return rest, true, nil
	case slices.Contains(s.AllNames, name):
		return "", false, nil
	}
	return "", false, fmt.Errorf("%w: no entry has the connection name of the reference, available names: %s", ErrSelection, quoteAll(s.AllNames))
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
	names := map[string][]string{}
	for _, e := range entries {
		names[e.storeType] = append(names[e.storeType], e.name)
	}
	for _, n := range names {
		slices.Sort(n)
	}
	out := make([]Scope, len(entries))
	for i, e := range entries {
		out[i] = Scope{Name: e.name, AllNames: names[e.storeType]}
	}
	return out
}
