// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package logs

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/open-policy-agent/opa/internal/deepcopy"
	"github.com/open-policy-agent/opa/util"
)

type maskOP string

const (
	maskOPRemove maskOP = "remove"
	maskOPUpsert maskOP = "upsert"

	partInput  = "input"
	partResult = "result"
)

var (
	errMaskInvalidObject = fmt.Errorf("mask upsert invalid object")
)

type maskRule struct {
	OP                maskOP      `json:"op"`
	Path              string      `json:"path"`
	Value             interface{} `json:"value"`
	escapedParts      []string
	modifyFullObj     bool
	failUndefinedPath bool
}

type maskRuleSet struct {
	OnRuleError  func(*maskRule, error)
	Rules        []*maskRule
	resultCopied bool
}

func (r maskRule) String() string {
	return "/" + strings.Join(r.escapedParts, "/")
}

type maskRuleOption func(*maskRule) error

func newMaskRule(path string, opts ...maskRuleOption) (*maskRule, error) {
	const (
		defaultOP                = maskOPRemove
		defaultFailUndefinedPath = false
	)

	if len(path) == 0 {
		return nil, fmt.Errorf("mask must be non-empty")
	} else if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("mask must be slash-prefixed")
	}

	parts := strings.Split(path[1:], "/")

	if parts[0] != partInput && parts[0] != partResult {
		return nil, fmt.Errorf("mask prefix not allowed: %v", parts[0])
	}

	escapedParts := make([]string, len(parts))
	for i := range parts {
		_, err := url.QueryUnescape(parts[i])
		if err != nil {
			return nil, err
		}

		escapedParts[i] = url.QueryEscape(parts[i])
	}

	modifyFullObj := false
	if len(escapedParts) == 1 {
		modifyFullObj = true
	}

	r := &maskRule{
		OP:                defaultOP,
		Path:              path,
		escapedParts:      escapedParts,
		failUndefinedPath: defaultFailUndefinedPath,
		modifyFullObj:     modifyFullObj,
	}

	for _, opt := range opts {
		if err := opt(r); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func withOP(op maskOP) maskRuleOption {
	return func(r *maskRule) error {

		var supportedMaskOPS = [...]maskOP{maskOPRemove, maskOPUpsert}
		for _, sOP := range supportedMaskOPS {
			if op == sOP {
				r.OP = op
				return nil
			}
		}

		return fmt.Errorf("mask op is not supported: %s", op)
	}
}

func withValue(val interface{}) maskRuleOption {
	return func(r *maskRule) error {
		r.Value = val
		return nil
	}
}

func withFailUndefinedPath() maskRuleOption {
	return func(r *maskRule) error {
		r.failUndefinedPath = true
		return nil
	}
}

func (r maskRule) Mask(event *EventV1) error {

	var maskObj *interface{}     // pointer to event Input|Result object
	var maskObjPtr **interface{} // pointer to the event Input|Result pointer itself

	switch p := r.escapedParts[0]; p {
	case partInput:
		if event.Input == nil {
			if r.failUndefinedPath {
				return errMaskInvalidObject
			}
			return nil
		}
		maskObj = event.Input
		maskObjPtr = &event.Input

	case partResult:
		if event.Result == nil {
			if r.failUndefinedPath {
				return errMaskInvalidObject
			}
			return nil
		}
		maskObj = event.Result
		maskObjPtr = &event.Result
	default:
		return fmt.Errorf("illegal path value: %s", p)
	}

	switch r.OP {
	case maskOPRemove:
		if r.modifyFullObj {
			*maskObjPtr = nil
		} else {

			parent, err := r.lookup(r.escapedParts[1:len(r.escapedParts)-1], *maskObj)
			if err != nil {
				if err == errMaskInvalidObject && r.failUndefinedPath {
					return err
				}
			}
			parentObj, ok := parent.(map[string]interface{})
			if !ok {
				return nil
			}

			fld := r.escapedParts[len(r.escapedParts)-1]
			if _, ok := parentObj[fld]; !ok {
				return nil
			}

			delete(parentObj, fld)

		}
		event.Erased = append(event.Erased, r.String())

	case maskOPUpsert:
		if r.modifyFullObj {
			*maskObjPtr = &r.Value
		} else {
			inputObj, ok := (*maskObj).(map[string]interface{})
			if !ok {
				return nil
			}

			if err := r.mkdirp(inputObj, r.escapedParts[1:len(r.escapedParts)], r.Value); err != nil {
				if r.failUndefinedPath {
					return err
				}

				return nil
			}
		}
		event.Masked = append(event.Masked, r.String())

	default:
		return fmt.Errorf("illegal mask op value: %s", r.OP)
	}

	return nil

}

func (r maskRule) lookup(p []string, node interface{}) (interface{}, error) {
	for i := 0; i < len(p); i++ {
		switch v := node.(type) {
		case map[string]interface{}:
			var ok bool
			if node, ok = v[p[i]]; !ok {
				return nil, errMaskInvalidObject
			}
		case []interface{}:
			idx, err := strconv.Atoi(p[i])
			if err != nil {
				return nil, errMaskInvalidObject
			} else if idx < 0 || idx >= len(v) {
				return nil, errMaskInvalidObject
			}
			node = v[idx]
		default:
			return nil, errMaskInvalidObject
		}
	}

	return node, nil
}

func (r maskRule) mkdirp(node map[string]interface{}, path []string, value interface{}) error {
	if len(path) == 0 {
		return nil
	}

	// create intermediate nodes
	for i := 0; i < len(path)-1; i++ {
		child, ok := node[path[i]]

		if !ok {
			child := map[string]interface{}{}
			node[path[i]] = child
			node = child
			continue
		}

		switch obj := child.(type) {
		case map[string]interface{}:
			node = obj
		default:
			return errMaskInvalidObject
		}

	}

	node[path[len(path)-1]] = value
	return nil
}

func newMaskRuleSet(rv interface{}, onRuleError func(*maskRule, error)) (*maskRuleSet, error) {
	bs, err := json.Marshal(rv)
	if err != nil {
		return nil, err
	}
	var mRuleSet = &maskRuleSet{
		OnRuleError: onRuleError,
	}
	var rawRules []interface{}

	if err := util.Unmarshal(bs, &rawRules); err != nil {
		return nil, err
	}

	for _, iface := range rawRules {

		switch v := iface.(type) {

		case string:
			// preserve default behavior of remove when
			// structured mask format is not provided
			rule, err := newMaskRule(v)
			if err != nil {
				return nil, err
			}

			mRuleSet.Rules = append(mRuleSet.Rules, rule)

		case map[string]interface{}:

			bs, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}

			rule := &maskRule{}

			if err := util.Unmarshal(bs, rule); err != nil {
				return nil, err
			}

			// use unmarshalled values to create new Mask Rule
			rule, err = newMaskRule(rule.Path, withOP(rule.OP), withValue(rule.Value))

			// TODO add withFailUndefinedPath() option based on
			//   A) new syntax in user defined mask rule
			//   B) passed in/global configuration option
			//   rule precedence A>B

			if err != nil {
				return nil, err
			}

			mRuleSet.Rules = append(mRuleSet.Rules, rule)

		default:
			return nil, fmt.Errorf("invalid mask rule format encountered: %T", v)
		}
	}

	return mRuleSet, nil
}

func (rs maskRuleSet) Mask(event *EventV1) {
	for _, mRule := range rs.Rules {
		// result must be deep copied if there are any mask rules
		// targeting it, to avoid modifying the result sent
		// to the consumer
		if mRule.escapedParts[0] == partResult && event.Result != nil && !rs.resultCopied {
			resultCopy := deepcopy.DeepCopy(*event.Result)
			event.Result = &resultCopy
			rs.resultCopied = true
		}
		err := mRule.Mask(event)
		if err != nil {
			rs.OnRuleError(mRule, err)
		}
	}
}
