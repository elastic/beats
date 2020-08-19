// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"net/url"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type dateCursor struct {
	log             *logp.Logger
	enabled         bool
	field           string
	url             url.URL
	urlField        string
	initialInterval time.Duration
	dateFormat      string

	value    string
	valueTpl *Template
}

func newDateCursorFromConfig(config config, log *logp.Logger) *dateCursor {
	c := &dateCursor{
		enabled: config.DateCursor.IsEnabled(),
		url:     *config.URL.URL,
	}

	if !c.enabled {
		return c
	}

	c.log = log
	c.field = config.DateCursor.Field
	c.urlField = config.DateCursor.URLField
	c.initialInterval = config.DateCursor.InitialInterval
	c.dateFormat = config.DateCursor.GetDateFormat()
	c.valueTpl = config.DateCursor.ValueTemplate

	return c
}

func (c *dateCursor) getURL() string {
	if !c.enabled {
		return c.url.String()
	}

	var dateStr string
	if c.value == "" {
		t := timeNow().UTC().Add(-c.initialInterval)
		dateStr = t.Format(c.dateFormat)
	} else {
		dateStr = c.value
	}

	q := c.url.Query()

	var value string
	if c.valueTpl == nil {
		value = dateStr
	} else {
		buf := new(bytes.Buffer)
		if err := c.valueTpl.Template.Execute(buf, dateStr); err != nil {
			return c.url.String()
		}
		value = buf.String()
	}

	q.Set(c.urlField, value)

	c.url.RawQuery = q.Encode()

	return c.url.String()
}

func (c *dateCursor) advance(m common.MapStr) {
	if c.field == "" {
		c.value = time.Now().UTC().Format(c.dateFormat)
		return
	}

	v, err := m.GetValue(c.field)
	if err != nil {
		c.log.Warnf("date_cursor field: %q", err)
		return
	}
	switch t := v.(type) {
	case string:
		_, err := time.Parse(c.dateFormat, t)
		if err != nil {
			c.log.Warn("date_cursor field does not have the expected layout")
			return
		}
		c.value = t
	default:
		c.log.Warn("date_cursor field must be a string, cursor will not advance")
		return
	}
}
