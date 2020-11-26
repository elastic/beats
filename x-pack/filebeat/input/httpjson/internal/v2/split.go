// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	errEmptyField    = errors.New("the requested field is emtpy")
	errEmptyTopField = errors.New("the requested top split field is emtpy")
)

type split struct {
	log        *logp.Logger
	targetInfo targetInfo
	kind       string
	transforms []basicTransform
	split      *split
	keepParent bool
	keyField   string
}

func newSplitResponse(cfg *splitConfig, log *logp.Logger) (*split, error) {
	if cfg == nil {
		return nil, nil
	}

	split, err := newSplit(cfg, log)
	if err != nil {
		return nil, err
	}

	if split.targetInfo.Type != targetBody {
		return nil, fmt.Errorf("invalid target type: %s", split.targetInfo.Type)
	}

	return split, nil
}

func newSplit(c *splitConfig, log *logp.Logger) (*split, error) {
	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return nil, err
	}

	ts, err := newBasicTransformsFromConfig(c.Transforms, responseNamespace, log)
	if err != nil {
		return nil, err
	}

	var s *split
	if c.Split != nil {
		s, err = newSplitResponse(c.Split, log)
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
		transforms: ts,
		split:      s,
	}, nil
}

func (s *split) run(ctx transformContext, resp *transformable, ch chan<- maybeMsg) error {
	return s.runChild(ctx, resp, ch, false)
}

func (s *split) runChild(ctx transformContext, resp *transformable, ch chan<- maybeMsg, isChild bool) error {
	respCpy := resp.clone()

	v, err := respCpy.body.GetValue(s.targetInfo.Name)
	if err != nil && err != common.ErrKeyNotFound {
		return err
	}

	switch s.kind {
	case "", splitTypeArr:
		if v == nil {
			s.log.Debug("array field is nil, sending main body")
			return s.sendEvent(ctx, respCpy, "", nil, ch, isChild)
		}

		arr, ok := v.([]interface{})
		if !ok {
			return fmt.Errorf("field %s needs to be an array to be able to split on it but it is %T", s.targetInfo.Name, v)
		}

		if len(arr) == 0 {
			s.log.Debug("array field is empty, sending main body")
			return s.sendEvent(ctx, respCpy, "", nil, ch, isChild)
		}

		for _, a := range arr {
			if err := s.sendEvent(ctx, respCpy, "", a, ch, isChild); err != nil {
				return err
			}
		}

		return nil
	case splitTypeMap:
		if v == nil {
			s.log.Debug("object field is nil, sending main body")
			return s.sendEvent(ctx, respCpy, "", nil, ch, isChild)
		}

		ms, ok := toMapStr(v)
		if !ok {
			return fmt.Errorf("field %s needs to be a map to be able to split on it but it is %T", s.targetInfo.Name, v)
		}

		if len(ms) == 0 {
			s.log.Debug("object field is empty, sending main body")
			return s.sendEvent(ctx, respCpy, "", nil, ch, isChild)
		}

		for k, v := range ms {
			if err := s.sendEvent(ctx, respCpy, k, v, ch, isChild); err != nil {
				return err
			}
		}

		return nil
	}

	return errors.New("invalid split type")
}

func toMapStr(v interface{}) (common.MapStr, bool) {
	var m common.MapStr
	if v == nil {
		return m, true
	}
	switch ts := v.(type) {
	case common.MapStr:
		m = ts
	case map[string]interface{}:
		m = common.MapStr(ts)
	default:
		return nil, false
	}
	return m, true
}

func (s *split) sendEvent(ctx transformContext, resp *transformable, key string, val interface{}, ch chan<- maybeMsg, isChild bool) error {
	if val == nil && !isChild && !s.keepParent {
		return errEmptyTopField
	}

	m, ok := toMapStr(val)
	if !ok {
		return errors.New("split can only be applied on object lists")
	}

	if s.keyField != "" && key != "" {
		_, _ = m.Put(s.keyField, key)
	}

	if s.keepParent {
		_, _ = resp.body.Put(s.targetInfo.Name, m)
	} else {
		resp.body = m
	}

	var err error
	for _, t := range s.transforms {
		resp, err = t.run(ctx, resp)
		if err != nil {
			return err
		}
	}

	if s.split != nil {
		return s.split.runChild(ctx, resp, ch, true)
	}

	ch <- maybeMsg{msg: resp.body.Clone()}

	if val == nil {
		return errEmptyField
	}

	return nil
}
