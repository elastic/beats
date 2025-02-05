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

package integration

import (
	"fmt"
	"regexp"
	"strings"
)

// OutputWatcher describes operations for watching output.
type OutputWatcher interface {
	// Inspect the line of the output and adjust the state accordingly.
	Inspect(string)
	// Observed is `true` if every expected output has been observed.
	Observed() bool
	// String is the string representation of the current state.
	// Describes what output is still expected.
	String() string
}

// NewStringWatcher creates a new output watcher that watches for a
// substring.
//
// The given string must be a substring of an output line
// to be marked as observed.
func NewStringWatcher(str string) OutputWatcher {
	return &stringWatcher{
		expecting: &str,
	}
}

type stringWatcher struct {
	expecting *string
}

func (w *stringWatcher) Inspect(line string) {
	if w.Observed() {
		return
	}
	if strings.Contains(line, *w.expecting) {
		w.expecting = nil
		return
	}
}

func (w *stringWatcher) Observed() bool {
	return w.expecting == nil
}

func (w *stringWatcher) String() string {
	if w.Observed() {
		return ""
	}
	return fmt.Sprintf("to have a substring %q", *w.expecting)
}

// NewRegexpWatcher create a new output watcher that watches for an
// output line to match the given regular expression.
func NewRegexpWatcher(expr *regexp.Regexp) OutputWatcher {
	return &regexpWatcher{
		expecting: expr,
	}
}

type regexpWatcher struct {
	expecting *regexp.Regexp
}

func (w *regexpWatcher) Inspect(line string) {
	if w.Observed() {
		return
	}
	if w.expecting.MatchString(line) {
		w.expecting = nil
	}
}

func (w *regexpWatcher) Observed() bool {
	return w.expecting == nil
}

func (w *regexpWatcher) String() string {
	if w.Observed() {
		return ""
	}
	return fmt.Sprintf("to match %s", w.expecting.String())
}

// NewInOrderWatcher creates a watcher that makes sure that the first
// watcher has `Observed() == true` then it moves on to the second,
// then third, etc.
//
// Reports overall state of all watchers on the list.
func NewInOrderWatcher(watchers []OutputWatcher) OutputWatcher {
	return &inOrderWatcher{
		watchers: watchers,
	}
}

type inOrderWatcher struct {
	watchers []OutputWatcher
}

func (w *inOrderWatcher) Inspect(line string) {
	if w.Observed() {
		return
	}
	w.watchers[0].Inspect(line)
	if w.watchers[0].Observed() {
		w.watchers = w.watchers[1:]
		return
	}
}

func (w *inOrderWatcher) Observed() bool {
	return len(w.watchers) == 0
}

func (w *inOrderWatcher) String() string {
	if w.Observed() {
		return ""
	}

	expectations := make([]string, 0, len(w.watchers))
	for _, watcher := range w.watchers {
		expectations = append(expectations, watcher.String())
	}
	return strings.Join(expectations, " -> ")
}

// NewOverallWatcher creates a watcher that reports an overall state
// of the list of other watchers.
//
// It's state marked as observed when all the nested watchers have
// `Observed() == true`.
func NewOverallWatcher(watchers []OutputWatcher) OutputWatcher {
	return &metaWatcher{
		active: watchers,
	}
}

type metaWatcher struct {
	active []OutputWatcher
}

func (w *metaWatcher) Inspect(line string) {
	var active []OutputWatcher
	for _, watcher := range w.active {
		watcher.Inspect(line)
		if !watcher.Observed() {
			active = append(active, watcher)
		}
	}
	w.active = active
}

func (w *metaWatcher) Observed() bool {
	return len(w.active) == 0
}

func (w *metaWatcher) String() string {
	if w.Observed() {
		return ""
	}

	expectations := make([]string, 0, len(w.active))
	for _, watcher := range w.active {
		expectations = append(expectations, watcher.String())
	}
	return " * " + strings.Join(expectations, "\n * ")
}
