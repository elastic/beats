// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type cursor struct {
	log *logp.Logger

	cfg cursorConfig

	state common.MapStr
}

func newCursor(cfg cursorConfig, log *logp.Logger) *cursor {
	return &cursor{cfg: cfg, log: log}
}

func (c *cursor) load(cursor *inputcursor.Cursor) {
	if c == nil || cursor == nil || cursor.IsNew() {
		c.log.Debug("new cursor: nothing loaded")
		return
	}

	if c.state == nil {
		c.state = common.MapStr{}
	}

	if err := cursor.Unpack(&c.state); err != nil {
		c.log.Errorf("Reset cursor state. Failed to read from registry: %v", err)
		return
	}

	c.log.Debugf("cursor loaded: %v", c.state)
}

func (c *cursor) update(trCtx *transformContext) {
	if c.cfg == nil {
		return
	}

	if c.state == nil {
		c.state = common.MapStr{}
	}

	for k, cfg := range c.cfg {
		v, _ := cfg.Value.Execute(trCtx, transformable{}, cfg.Default, c.log)
		if v != "" || !cfg.mustIgnoreEmptyValue() {
			_, _ = c.state.Put(k, v)
			c.log.Debugf("cursor.%s stored with %s", k, v)
		}
	}
}

func (c *cursor) clone() common.MapStr {
	if c == nil || c.state == nil {
		return common.MapStr{}
	}
	return c.state.Clone()
}
