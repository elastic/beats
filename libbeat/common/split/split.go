// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package split

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	errEmptyField          = errors.New("the requested field is empty")
	errEmptyRootField      = errors.New("the requested root field is empty")
	errExpectedSplitArr    = errors.New("split was expecting field to be an array")
	errExpectedSplitObj    = errors.New("split was expecting field to be an object")
	errExpectedSplitString = errors.New("split was expecting field to be a string")
)

type splitConfig struct {
	Target string `config:"target" validation:"required"`
	// Type   string `config:"type"`
	Split      *splitConfig `config:"split"`
	KeepParent bool         `config:"keep_parent"`
	// KeyField   string `config:"key_field"`
	// DelimiterString  string       `config:"delimiter"`
	IgnoreEmptyValue bool `config:"ignore_empty_value"`
}

// split is a split processor chain element. Split processing is executed
// by applying elements of the chain's linked list to an input until completed
// or an error state is encountered.
type split struct {
	log    *logp.Logger
	target string
	// kind   string
	child            *split
	keepParent       bool
	ignoreEmptyValue bool
	// keyField         string
	isRoot bool
	// delimiter        string
}
type maybeMsg struct {
	err error
	msg common.MapStr
}

func (e maybeMsg) failed() bool { return e.err != nil }

func (e maybeMsg) Error() string { return e.err.Error() }

// newSplit returns a new split based on the provided config and
// logging to the provided logger.
func newSplit(c *splitConfig, log *logp.Logger) (*split, error) {
	var s *split
	if c.Split != nil {
		s, _ = newSplit(c.Split, log)
		// if err != nil {
		// 	return nil, err
		// }
	}

	return &split{
		log:    log,
		target: c.Target,
		// kind:             c.Type,
		keepParent:       c.KeepParent,
		ignoreEmptyValue: c.IgnoreEmptyValue,
		// keyField:         c.KeyField,
		// delimiter:        c.DelimiterString,
		child: s,
	}, nil
}

func (s *split) startSplit(raw json.RawMessage) (<-chan maybeMsg, error) {
	ch := make(chan maybeMsg)
	var jsonObject common.MapStr
	if err := json.Unmarshal(raw, &jsonObject); err != nil {
		return nil, err
	}

	go func() {
		defer close(ch)
		if err := s.run(jsonObject, ch); err != nil {
			switch err {
			case errEmptyField:
				// nothing else to send for this page
				s.log.Debug("split operation finished")
				return
			case errEmptyRootField:
				// root field not found, most likely the response is empty
				s.log.Debug(err)
				return
			default:
				s.log.Debug("split operation failed")
				ch <- maybeMsg{err: err}
				return
			}
		}
	}()

	return ch, nil
}

// run runs the split operation on the contents of resp, sending successive
// split results on ch. ctx is passed to transforms that are called during
// the split.
func (s *split) run(jsonObject common.MapStr, ch chan<- maybeMsg) error {
	// var jsonObject common.MapStr
	// if err := json.Unmarshal(raw, &jsonObject); err != nil {
	// 	return err
	// }
	return s.split(jsonObject, ch)
}

// split recursively executes the split processor chain.
func (s *split) split(root common.MapStr, ch chan<- maybeMsg) error {
	v, err := root.GetValue(s.target)
	s.log.Info(s.target)
	s.log.Info(v)
	if err != nil && err != common.ErrKeyNotFound {
		return err
	}

	if v == nil {
		if s.ignoreEmptyValue {
			if s.child != nil {
				return s.child.split(root, ch)
			}
			return nil
		}
		if s.isRoot {
			return errEmptyRootField
		}
		ch <- maybeMsg{msg: root}
		return errEmptyField
	}

	varr, ok := v.([]interface{})
	if !ok {
		return errExpectedSplitArr
	}

	if len(varr) == 0 {
		if s.ignoreEmptyValue {
			if s.child != nil {
				return s.child.split(root, ch)
			}
			return nil
		}
		if s.isRoot {
			return errEmptyRootField
		}
		ch <- maybeMsg{msg: root}
		return errEmptyField
	}

	for _, e := range varr {
		if err := s.sendMessage(root, e, ch); err != nil {
			s.log.Debug(err)
		}
	}

	return nil

	// return errors.New("unknown split type")
}

// sendMessage sends an array or map split result value, v, on ch after performing
// any necessary transformations. If key is "", the value is an element of an array.
func (s *split) sendMessage(root common.MapStr, v interface{}, ch chan<- maybeMsg) error {
	obj, ok := toInterfaceStr(v)
	if !ok {
		return errExpectedSplitObj
	}

	clone := root.Clone()

	if s.keepParent {
		_, _ = clone.Put(s.target, obj)
	} else {
		obj, ok := toMapStr(v)
		if !ok {
			return errExpectedSplitObj
		}
		clone = obj
	}

	if s.child != nil {
		return s.child.split(clone, ch)
	}
	// data, _ := json.Marshal(clone)
	ch <- maybeMsg{msg: clone}

	return nil
}

func toMapStr(v interface{}) (common.MapStr, bool) {
	if v == nil {
		return common.MapStr{}, false
	}
	switch t := v.(type) {
	case common.MapStr:
		return t, true
	case map[string]interface{}:
		return common.MapStr(t), true
	case string:
		temp := make(map[string]interface{})
		temp["data"] = t
		return common.MapStr(temp), true
	}
	return common.MapStr{}, false
}

func toInterfaceStr(v interface{}) (interface{}, bool) {
	if v == nil {
		return common.MapStr{}, false
	}
	switch t := v.(type) {
	case common.MapStr:
		return t, true
	case map[string]interface{}:
		return common.MapStr(t), true
	case string:
		return t, true
	}
	return common.MapStr{}, false
}
