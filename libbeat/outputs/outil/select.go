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

package outil

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/conditions"
)

// Selector is used to produce a string based on the contents of a Beats event.
// A selector supports multiple rules that need to be configured.
type Selector struct {
	sel SelectorExpr
}

// SelectorExpr represents an expression object that can be composed with other
// expressions in order to build a Selector.
type SelectorExpr interface {
	sel(evt *beat.Event) (string, error)
}

type emptySelector struct{}

type listSelector struct {
	selectors []SelectorExpr
}

type condSelector struct {
	s    SelectorExpr
	cond conditions.Condition
}

type constSelector struct {
	s string
}

type fmtSelector struct {
	f         fmtstr.EventFormatString
	otherwise string
	selCase   SelectorCase
}

type mapSelector struct {
	from      SelectorExpr
	otherwise string
	to        map[string]string
}

var nilSelector SelectorExpr = &emptySelector{}

// MakeSelector creates a selector from a set of selector expressions.
func MakeSelector(es ...SelectorExpr) Selector {
	switch len(es) {
	case 0:
		return Selector{nilSelector}
	case 1:
		return Selector{es[0]}
	default:
		return Selector{ConcatSelectorExpr(es...)}
	}
}

// Select runs configured selector against the current event.
// If no matching selector is found, an empty string is returned.
// It's up to the caller to decide if an empty string is an error
// or an expected result.
func (s Selector) Select(evt *beat.Event) (string, error) {
	return s.sel.sel(evt)
}

func (s Selector) IsAlias() bool { return false }

// IsEmpty checks if the selector is not configured and will always return an empty string.
func (s Selector) IsEmpty() bool {
	return s.sel == nilSelector || s.sel == nil
}

// IsConst checks if the selector will always return the same string.
func (s Selector) IsConst() bool {
	if s.sel == nilSelector {
		return true
	}

	_, ok := s.sel.(*constSelector)
	return ok
}

// BuildSelectorFromConfig creates a selector from a configuration object.
func BuildSelectorFromConfig(
	cfg *common.Config,
	settings Settings,
) (Selector, error) {
	var sel []SelectorExpr

	key := settings.Key
	multiKey := settings.MultiKey
	found := false

	if cfg.HasField(multiKey) {
		found = true
		sub, err := cfg.Child(multiKey, -1)
		if err != nil {
			return Selector{}, err
		}

		var table []*common.Config
		if err := sub.Unpack(&table); err != nil {
			return Selector{}, err
		}

		for _, config := range table {
			action, err := buildSingle(config, key, settings.Case)
			if err != nil {
				return Selector{}, err
			}

			if action != nilSelector {
				sel = append(sel, action)
			}
		}
	}

	if settings.EnableSingleOnly && cfg.HasField(key) {
		found = true

		// expect event-format-string
		str, err := cfg.String(key, -1)
		if err != nil {
			return Selector{}, err
		}

		fmtstr, err := fmtstr.CompileEvent(str)
		if err != nil {
			return Selector{}, fmt.Errorf("%v in %v", err, cfg.PathOf(key))
		}

		fmtsel, err := FmtSelectorExpr(fmtstr, "", settings.Case)
		if err != nil {
			return Selector{}, fmt.Errorf("%v in %v", err, cfg.PathOf(key))
		}

		if fmtsel != nilSelector {
			sel = append(sel, fmtsel)
		}
	}

	if settings.FailEmpty && !found {
		if settings.EnableSingleOnly {
			return Selector{}, fmt.Errorf("missing required '%v' or '%v' in %v",
				key, multiKey, cfg.Path())
		}

		return Selector{}, fmt.Errorf("missing required '%v' in %v",
			multiKey, cfg.Path())
	}

	return MakeSelector(sel...), nil
}

// EmptySelectorExpr create a selector expression that returns an empty string.
func EmptySelectorExpr() SelectorExpr {
	return nilSelector
}

// ConstSelectorExpr creates a selector expression that always returns the configured string.
func ConstSelectorExpr(s string, selCase SelectorCase) SelectorExpr {
	if s == "" {
		return EmptySelectorExpr()
	}
	return &constSelector{selCase.apply(s)}
}

// FmtSelectorExpr creates a selector expression using a format string. If the
// event can not be applied the default fallback constant string will be returned.
func FmtSelectorExpr(fmt *fmtstr.EventFormatString, fallback string, selCase SelectorCase) (SelectorExpr, error) {
	if fmt.IsConst() {
		str, err := fmt.Run(nil)
		if err != nil {
			return nil, err
		}
		if str == "" {
			str = fallback
		}
		return ConstSelectorExpr(str, selCase), nil
	}

	return &fmtSelector{*fmt, selCase.apply(fallback), selCase}, nil
}

// ConcatSelectorExpr combines multiple expressions that are run one after the other.
// The first expression that returns a string wins.
func ConcatSelectorExpr(s ...SelectorExpr) SelectorExpr {
	return &listSelector{s}
}

// ConditionalSelectorExpr executes the given expression only if the event
// matches the given condition.
func ConditionalSelectorExpr(
	s SelectorExpr,
	cond conditions.Condition,
) SelectorExpr {
	return &condSelector{s, cond}
}

// LookupSelectorExpr replaces the produced string with an table entry.
// If there is no entry in the table the default fallback string will be reported.
func LookupSelectorExpr(
	evtfmt *fmtstr.EventFormatString,
	table map[string]string,
	fallback string,
	selCase SelectorCase,
) (SelectorExpr, error) {
	if evtfmt.IsConst() {
		str, err := evtfmt.Run(nil)
		if err != nil {
			return nil, err
		}

		str = table[selCase.apply(str)]
		if str == "" {
			str = fallback
		}
		return ConstSelectorExpr(str, selCase), nil
	}

	return &mapSelector{
		from:      &fmtSelector{f: *evtfmt},
		to:        table,
		otherwise: fallback,
	}, nil
}

func copyTable(selCase SelectorCase, table map[string]string) map[string]string {
	tmp := make(map[string]string, len(table))
	for k, v := range table {
		tmp[selCase.apply(k)] = selCase.apply(v)
	}
	return tmp
}

func buildSingle(cfg *common.Config, key string, selCase SelectorCase) (SelectorExpr, error) {
	// TODO: check for unknown fields

	// 1. extract required key-word handler
	if !cfg.HasField(key) {
		return nil, fmt.Errorf("missing %v", cfg.PathOf(key))
	}

	str, err := cfg.String(key, -1)
	if err != nil {
		return nil, err
	}

	evtfmt, err := fmtstr.CompileEvent(str)
	if err != nil {
		return nil, fmt.Errorf("%v in %v", err, cfg.PathOf(key))
	}

	// 2. extract optional `default` value
	var otherwise string
	if cfg.HasField("default") {
		tmp, err := cfg.String("default", -1)
		if err != nil {
			return nil, err
		}
		otherwise = selCase.apply(tmp)
	}

	// 3. extract optional `mapping`
	mapping := struct {
		Table map[string]string `config:"mappings"`
	}{nil}
	if cfg.HasField("mappings") {
		if err := cfg.Unpack(&mapping); err != nil {
			return nil, err
		}
	}

	// 4. extract conditional
	var cond conditions.Condition
	if cfg.HasField("when") {
		sub, err := cfg.Child("when", -1)
		if err != nil {
			return nil, err
		}

		condConfig := conditions.Config{}
		if err := sub.Unpack(&condConfig); err != nil {
			return nil, err
		}

		tmp, err := conditions.NewCondition(&condConfig)
		if err != nil {
			return nil, err
		}

		cond = tmp
	}

	// 5. build selector from available fields
	var sel SelectorExpr
	if len(mapping.Table) > 0 {
		sel, err = LookupSelectorExpr(evtfmt, copyTable(selCase, mapping.Table), otherwise, selCase)
	} else {
		sel, err = FmtSelectorExpr(evtfmt, otherwise, selCase)
	}
	if err != nil {
		return nil, err
	}

	if cond != nil && sel != nilSelector {
		sel = ConditionalSelectorExpr(sel, cond)
	}

	return sel, nil
}

func (s *emptySelector) sel(evt *beat.Event) (string, error) {
	return "", nil
}

func (s *listSelector) sel(evt *beat.Event) (string, error) {
	for _, sub := range s.selectors {
		n, err := sub.sel(evt)
		if err != nil { // TODO: try
			return n, err
		}

		if n != "" {
			return n, nil
		}
	}

	return "", nil
}

func (s *condSelector) sel(evt *beat.Event) (string, error) {
	if !s.cond.Check(evt) {
		return "", nil
	}
	return s.s.sel(evt)
}

func (s *constSelector) sel(_ *beat.Event) (string, error) {
	return s.s, nil
}

func (s *fmtSelector) sel(evt *beat.Event) (string, error) {
	n, err := s.f.Run(evt)
	if err != nil {
		// err will be set if not all keys present in event ->
		// return empty selector result and ignore error
		return s.otherwise, nil
	}

	if n == "" {
		return s.otherwise, nil
	}
	return s.selCase.apply(n), nil
}

func (s *mapSelector) sel(evt *beat.Event) (string, error) {
	n, err := s.from.sel(evt)
	if err != nil {
		if s.otherwise == "" {
			return "", err
		}
		return s.otherwise, nil
	}

	if n == "" {
		return s.otherwise, nil
	}

	n = s.to[n]
	if n == "" {
		return s.otherwise, nil
	}
	return n, nil
}
