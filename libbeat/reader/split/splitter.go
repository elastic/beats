// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package split

import (
	"encoding/json"
	"errors"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	errEmptyField          = errors.New("the requested field is empty")
	errEmptyRootField      = errors.New("the requested root field is empty")
	errExpectedSplitArr    = errors.New("split was expecting field to be an array")
	errExpectedSplitObj    = errors.New("split was expecting field to be an object")
	errExpectedSplitString = errors.New("split was expecting field to be a string")
)

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
	IsRoot bool
	// delimiter        string
}
type maybeMsg struct {
	err error
	Msg mapstr.M
}

func (e maybeMsg) Failed() bool { return e.err != nil }

func (e maybeMsg) Error() string { return e.err.Error() }

// newSplit returns a new split based on the provided config and
// logging to the provided logger.
func NewSplit(c *Config, log *logp.Logger) (*split, error) {
	var s *split
	if c.Split != nil {
		s, _ = NewSplit(c.Split, log)
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

func (s *split) StartSplit(raw json.RawMessage) (<-chan maybeMsg, error) {
	ch := make(chan maybeMsg)
	var jsonObject mapstr.M
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
func (s *split) run(jsonObject mapstr.M, ch chan<- maybeMsg) error {
	// var jsonObject mapstr.M
	// if err := json.Unmarshal(raw, &jsonObject); err != nil {
	// 	return err
	// }
	return s.split(jsonObject, ch)
}

// split recursively executes the split processor chain.
func (s *split) split(root mapstr.M, ch chan<- maybeMsg) error {
	v, err := root.GetValue(s.target)
	if err != nil && err != mapstr.ErrKeyNotFound {
		return err
	}

	if v == nil {
		if s.ignoreEmptyValue {
			if s.child != nil {
				return s.child.split(root, ch)
			}
			return nil
		}
		if s.IsRoot {
			return errEmptyRootField
		}
		ch <- maybeMsg{Msg: root}
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
		if s.IsRoot {
			return errEmptyRootField
		}
		ch <- maybeMsg{Msg: root}
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
func (s *split) sendMessage(root mapstr.M, v interface{}, ch chan<- maybeMsg) error {
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
	ch <- maybeMsg{Msg: clone}

	return nil
}

func toMapStr(v interface{}) (mapstr.M, bool) {
	if v == nil {
		return mapstr.M{}, false
	}
	switch t := v.(type) {
	case mapstr.M:
		return t, true
	case map[string]interface{}:
		return mapstr.M(t), true
	case string:
		temp := make(map[string]interface{})
		temp["data"] = t
		return mapstr.M(temp), true
	}
	return mapstr.M{}, false
}

func toInterfaceStr(v interface{}) (interface{}, bool) {
	if v == nil {
		return mapstr.M{}, false
	}
	switch t := v.(type) {
	case mapstr.M:
		return t, true
	case map[string]interface{}:
		return mapstr.M(t), true
	case string:
		return t, true
	}
	return mapstr.M{}, false
}
