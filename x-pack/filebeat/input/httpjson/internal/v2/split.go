// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"errors"
	"fmt"
	"strings"

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

type split struct {
	log        *logp.Logger
	targetInfo targetInfo
	kind       string
	transforms []basicTransform
	child      *split
	keepParent bool
	keyField   string
	isRoot     bool
	delimiter  string
}

func newSplitResponse(cfg *splitConfig, log *logp.Logger) (*split, error) {
	if cfg == nil {
		return nil, nil
	}

	split, err := newSplit(cfg, log)
	if err != nil {
		return nil, err
	}
	// we want to be able to identify which split is the root of the chain
	split.isRoot = true
	return split, nil
}

func newSplit(c *splitConfig, log *logp.Logger) (*split, error) {
	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return nil, err
	}

	if ti.Type != targetBody {
		return nil, fmt.Errorf("invalid target type: %s", ti.Type)
	}

	ts, err := newBasicTransformsFromConfig(c.Transforms, responseNamespace, log)
	if err != nil {
		return nil, err
	}

	var s *split
	if c.Split != nil {
		s, err = newSplit(c.Split, log)
		if err != nil {
			return nil, err
		}
	}

	return &split{
		log:        log,
		targetInfo: ti,
		kind:       c.Type,
		keepParent: c.KeepParent,
		keyField:   c.KeyField,
		delimiter:  c.DelimiterString,
		transforms: ts,
		child:      s,
	}, nil
}

func (s *split) run(ctx *transformContext, resp transformable, ch chan<- maybeMsg) error {
	root := resp.body()
	return s.split(ctx, root, ch)
}

func (s *split) split(ctx *transformContext, root common.MapStr, ch chan<- maybeMsg) error {
	v, err := root.GetValue(s.targetInfo.Name)
	if err != nil && err != common.ErrKeyNotFound {
		return err
	}

	if v == nil {
		if s.isRoot {
			return errEmptyRootField
		}
		ch <- maybeMsg{msg: root}
		return errEmptyField
	}

	switch s.kind {
	case "", splitTypeArr:
		varr, ok := v.([]interface{})
		if !ok {
			return errExpectedSplitArr
		}

		if len(varr) == 0 {
			if s.isRoot {
				return errEmptyRootField
			}
			ch <- maybeMsg{msg: root}
			return errEmptyField
		}

		for _, e := range varr {
			if err := s.sendMessage(ctx, root, "", e, ch); err != nil {
				s.log.Debug(err)
			}
		}

		return nil
	case splitTypeMap:
		vmap, ok := toMapStr(v)
		if !ok {
			return errExpectedSplitObj
		}

		if len(vmap) == 0 {
			if s.isRoot {
				return errEmptyRootField
			}
			ch <- maybeMsg{msg: root}
			return errEmptyField
		}

		for k, e := range vmap {
			if err := s.sendMessage(ctx, root, k, e, ch); err != nil {
				s.log.Debug(err)
			}
		}

		return nil
	case splitTypeString:
		vstr, ok := v.(string)
		if !ok {
			return errExpectedSplitString
		}

		if len(vstr) == 0 {
			if s.isRoot {
				return errEmptyRootField
			}
			ch <- maybeMsg{msg: root}
			return errEmptyField
		}
		for _, substr := range strings.Split(vstr, s.delimiter) {
			if err := s.sendMessageSplitString(ctx, root, substr, ch); err != nil {
				s.log.Debug(err)
			}
		}

		return nil
	}

	return errors.New("unknown split type")
}

func (s *split) sendMessage(ctx *transformContext, root common.MapStr, key string, v interface{}, ch chan<- maybeMsg) error {
	obj, ok := toMapStr(v)
	if !ok {
		return errExpectedSplitObj
	}

	clone := root.Clone()

	if s.keyField != "" && key != "" {
		_, _ = obj.Put(s.keyField, key)
	}

	if s.keepParent {
		_, _ = clone.Put(s.targetInfo.Name, obj)
	} else {
		clone = obj
	}

	tr := transformable{}
	tr.setBody(clone)

	var err error
	for _, t := range s.transforms {
		tr, err = t.run(ctx, tr)
		if err != nil {
			return err
		}
	}

	if s.child != nil {
		return s.child.split(ctx, clone, ch)
	}

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
	}
	return common.MapStr{}, false
}

func (s *split) sendMessageSplitString(ctx *transformContext, root common.MapStr, v string, ch chan<- maybeMsg) error {
	clone := root.Clone()
	_, _ = clone.Put(s.targetInfo.Name, v)

	tr := transformable{}
	tr.setBody(clone)

	var err error
	for _, t := range s.transforms {
		tr, err = t.run(ctx, tr)
		if err != nil {
			return err
		}
	}

	if s.child != nil {
		return s.child.split(ctx, clone, ch)
	}

	ch <- maybeMsg{msg: clone}

	return nil
}
