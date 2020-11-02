// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
)

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
	case splitTypeArr:
		arr, ok := v.([]interface{})
		if !ok {
			return fmt.Errorf("field %s needs to be an array to be able to split on it", s.targetInfo.Name)
		}
		for _, a := range arr {
			m, ok := a.(map[string]interface{})
			if !ok {
				// TODO
			}

			if s.keepParent {
				_, _ = respCpy.body.Put(s.targetInfo.Name, m)
			} else {
				respCpy.body = common.MapStr(m)
			}

			if s.split != nil {
				return s.split.run(ctx, respCpy, ch)
			}

			event, err := makeEvent(respCpy.body)
			if err != nil {
				return err
			}

			ch <- maybeEvent{event: event}
		}

		return nil
	case splitTypeMap:
		m, ok := v.(map[string]interface{})
		if !ok {
			return fmt.Errorf("field %s needs to be a map to be able to split on it", s.targetInfo.Name)
		}
		for k, mm := range m {
			v, ok := mm.(map[string]interface{})
			if !ok {
				// TODO
			}

			vv := common.MapStr(v)
			if s.keyField != "" {
				_, _ = vv.Put(s.keyField, k)
			}

			if s.keepParent {
				_, _ = respCpy.body.Put(s.targetInfo.Name, vv)
			} else {
				respCpy.body = vv
			}

			if s.split != nil {
				return s.split.run(ctx, respCpy, ch)
			}

			event, err := makeEvent(respCpy.body)
			if err != nil {
				return err
			}

			ch <- maybeEvent{event: event}
		}

		return nil
	}

	return errors.New("invalid split type")
}
