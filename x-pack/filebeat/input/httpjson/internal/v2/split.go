// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
)

var errEmtpyField = errors.New("the requested field is emtpy")

type split struct {
	targetInfo targetInfo
	kind       string
	transforms []basicTransform
	split      *split
	keepParent bool
	keyField   string
}

func newSplitResponse(cfg *splitConfig) (*split, error) {
	if cfg == nil {
		return nil, nil
	}

	split, err := newSplit(cfg)
	if err != nil {
		return nil, err
	}

	if split.targetInfo.Type != targetBody {
		return nil, fmt.Errorf("invalid target type: %s", split.targetInfo.Type)
	}

	return split, nil
}

func newSplit(c *splitConfig) (*split, error) {
	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return nil, err
	}

	ts, err := newBasicTransformsFromConfig(c.Transforms, responseNamespace)
	if err != nil {
		return nil, err
	}

	var s *split
	if c.Split != nil {
		s, err = newSplitResponse(c.Split)
		if err != nil {
			return nil, err
		}
	}

	return &split{
		targetInfo: ti,
		kind:       c.Type,
		keepParent: c.KeepParent,
		keyField:   c.KeyField,
		transforms: ts,
		split:      s,
	}, nil
}

func (s *split) run(ctx transformContext, resp *transformable, ch chan<- maybeEvent) error {
	respCpy := resp.clone()
	var err error
	for _, t := range s.transforms {
		respCpy, err = t.run(ctx, respCpy)
		if err != nil {
			return err
		}
	}

	v, err := respCpy.body.GetValue(s.targetInfo.Name)
	if err != nil && err != common.ErrKeyNotFound {
		return err
	}

	switch s.kind {
	case "", splitTypeArr:
		arr, ok := v.([]interface{})
		if !ok {
			return fmt.Errorf("field %s needs to be an array to be able to split on it but it is %T", s.targetInfo.Name, v)
		}

		if len(arr) == 0 {
			return errEmtpyField
		}

		for _, a := range arr {
			m, ok := toMapStr(a)
			if !ok {
				return errors.New("split can only be applied on object lists")
			}

			if err := s.sendEvent(ctx, respCpy, m, ch); err != nil {
				return err
			}
		}

		return nil
	case splitTypeMap:
		if v == nil {
			return errEmtpyField
		}

		ms, ok := toMapStr(v)
		if !ok {
			return fmt.Errorf("field %s needs to be a map to be able to split on it but it is %T", s.targetInfo.Name, v)
		}

		if len(ms) == 0 {
			return errEmtpyField
		}

		for k, v := range ms {
			m, ok := toMapStr(v)
			if !ok {
				return errors.New("split can only be applied on object lists")
			}
			if s.keyField != "" {
				_, _ = m.Put(s.keyField, k)
			}
			if err := s.sendEvent(ctx, respCpy, m, ch); err != nil {
				return err
			}
		}

		return nil
	}

	return errors.New("invalid split type")
}

func toMapStr(v interface{}) (common.MapStr, bool) {
	var m common.MapStr
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

func (s *split) sendEvent(ctx transformContext, resp *transformable, m common.MapStr, ch chan<- maybeEvent) error {
	if s.keepParent {
		_, _ = resp.body.Put(s.targetInfo.Name, m)
	} else {
		resp.body = m
	}

	if s.split != nil {
		return s.split.run(ctx, resp, ch)
	}

	event, err := makeEvent(resp.body)
	if err != nil {
		return err
	}

	ch <- maybeEvent{event: event}

	*ctx.lastEvent = event

	return nil
}
