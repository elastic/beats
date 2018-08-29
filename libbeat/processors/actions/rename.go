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

package actions

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type renameFields struct {
	config renameFieldsConfig

	renamers []fieldRenamer
}

type renameFieldsConfig struct {
	Fields        []fromTo `config:"fields"`
	IgnoreMissing bool     `config:"ignore_missing"`
	FailOnError   bool     `config:"fail_on_error"`
}

type fieldRenamer func(ctx *renameCtx, evt *beat.Event) error

type renameCtx struct {
	cfg   *renameFieldsConfig
	event beat.Event

	backedUp struct {
		meta   bool
		fields bool
	}
}

type fromTo struct {
	From string `config:"from"`
	To   string `config:"to"`
}

type fieldAccessor interface {
	IsMeta() bool
	IsFields() bool
	Has(evt *beat.Event) bool
	Get(evt *beat.Event) (interface{}, error)
	Put(evt *beat.Event, value interface{}) error
	Delete(evt *beat.Event) error
}

type timestampFieldAccessor struct{}

type metaFieldAccessor struct {
	field string
}

type eventFieldAccessor struct {
	field string
}

var errDelTimestamp = errors.New("can not delete @timestamp")

var _timestampFieldAccessor = (*timestampFieldAccessor)(nil)

func init() {
	processors.RegisterPlugin("rename",
		configChecked(newRenameFields,
			requireFields("fields")))
}

func newRenameFields(c *common.Config) (processors.Processor, error) {
	config := renameFieldsConfig{
		IgnoreMissing: false,
		FailOnError:   true,
	}
	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the rename configuration: %s", err)
	}

	renamers := make([]fieldRenamer, len(config.Fields))
	for i, fields := range config.Fields {
		renamers[i] = makeFieldRenamer(fields)
	}

	f := &renameFields{
		config:   config,
		renamers: renamers,
	}
	return f, nil
}

func (f *renameFields) Run(event *beat.Event) (*beat.Event, error) {
	var err error
	ctx := renameCtx{cfg: &f.config}
	if f.config.FailOnError {
		ctx.event = *event
	}

	for _, renamer := range f.renamers {
		err = renamer(&ctx, event)
		if err != nil && f.config.FailOnError {
			break
		}
	}

	if err != nil && f.config.FailOnError {
		// processing failed, restore backup:
		event.Timestamp = ctx.event.Timestamp
		if ctx.backedUp.meta {
			event.Meta = ctx.event.Meta
		}
		if ctx.backedUp.fields {
			event.Fields = ctx.event.Fields
		}
		return event, err
	}

	return event, nil
}

func (f *renameFields) String() string {
	return "rename=" + fmt.Sprintf("%+v", f.config.Fields)
}

func makeFieldRenamer(fields fromTo) fieldRenamer {
	from := makeAccessor(fields.From)
	to := makeAccessor(fields.To)

	backupMeta := from.IsMeta() || to.IsMeta()
	backupFields := from.IsFields() || to.IsFields()

	return func(ctx *renameCtx, evt *beat.Event) error {
		if !from.Has(evt) {
			return fmt.Errorf("target field %s already exists, drop or rename this field first", to)
		}

		value, err := from.Get(evt)
		if err != nil {
			// Ignore ErrKeyNotFound errors
			if ctx.cfg.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
				return nil
			}
			return fmt.Errorf("could not fetch value for key: %s, Error: %s", from, err)
		}

		// lazily backup fields by cloning original contents
		if ctx.cfg.FailOnError {
			if backupMeta && !ctx.backedUp.meta {
				ctx.event.Meta = evt.Meta.Clone()
				ctx.backedUp.meta = true
			}
			if backupFields && !ctx.backedUp.fields {
				ctx.event.Fields = evt.Fields.Clone()
				ctx.backedUp.fields = true
			}
		}

		// Deletion must happen first to support cases where a becomes a.b
		err = from.Delete(evt)
		if err != nil {
			return fmt.Errorf("could not delete key: %s,  %+v", from, err)
		}

		err = to.Put(evt, value)
		if err != nil {
			return fmt.Errorf("could not put value: %s: %v, %+v", to, value, err)
		}

		return nil
	}
}

func makeAccessor(field string) fieldAccessor {
	switch {
	case field == "@timestamp":
		return _timestampFieldAccessor
	case isMetadata(field):
		return newMetaFieldAccessor(field)
	default:
		return newEventFieldAccessor(field)
	}
}

func newMetaFieldAccessor(path string) *metaFieldAccessor {
	const prefix = "@metadata."
	if strings.HasPrefix(path, prefix) {
		path = path[len(prefix):]
	} else if path == "@metadata" {
		path = ""
	}
	return &metaFieldAccessor{path}
}

func (a *metaFieldAccessor) IsMeta() bool { return true }

func (a *metaFieldAccessor) IsFields() bool { return false }

func (a *metaFieldAccessor) Has(evt *beat.Event) bool {
	return (evt.Meta != nil) && (a.field == "" || hasField(evt.Meta, a.field))
}

func (a *metaFieldAccessor) Get(evt *beat.Event) (interface{}, error) {
	if a.field == "" {
		return evt.Meta, nil
	}
	if evt.Meta == nil {
		return nil, nil
	}
	return evt.Meta.GetValue(a.field)
}

func (a *metaFieldAccessor) Put(evt *beat.Event, value interface{}) error {
	if a.field == "" {
		m, ok := value.(common.MapStr)
		if !ok {
			return fmt.Errorf("object required to set @metadata, but got %T", value)
		}
		evt.Meta = m
	}

	if evt.Meta == nil {
		evt.Meta = common.MapStr{}
	}
	_, err := evt.Meta.Put(a.field, value)
	return err
}

func (a *metaFieldAccessor) Delete(evt *beat.Event) error {
	if a.field == "" {
		evt.Meta = nil
		return nil
	}
	if evt.Meta == nil {
		return nil
	}
	return evt.Meta.Delete(a.field)
}

func newEventFieldAccessor(path string) *eventFieldAccessor {
	return &eventFieldAccessor{path}
}

func (a *eventFieldAccessor) IsMeta() bool { return false }

func (a *eventFieldAccessor) IsFields() bool { return true }

func (a *eventFieldAccessor) Has(evt *beat.Event) bool {
	return hasField(evt.Fields, a.field)
}

func (a *eventFieldAccessor) Get(evt *beat.Event) (interface{}, error) {
	return evt.Fields.GetValue(a.field)
}

func (a *eventFieldAccessor) Put(evt *beat.Event, value interface{}) error {
	_, err := evt.Fields.Put(a.field, value)
	return err
}

func (a *eventFieldAccessor) Delete(evt *beat.Event) error {
	return evt.Fields.Delete(a.field)
}

func (a *timestampFieldAccessor) IsMeta() bool             { return false }
func (a *timestampFieldAccessor) IsFields() bool           { return false }
func (a *timestampFieldAccessor) Has(evt *beat.Event) bool { return true }

func (a *timestampFieldAccessor) Get(evt *beat.Event) (interface{}, error) {
	return evt.Timestamp, nil
}

func (a *timestampFieldAccessor) Put(evt *beat.Event, value interface{}) error {
	switch ts := value.(type) {
	case time.Time:
		evt.Timestamp = ts
	case common.Time:
		evt.Timestamp = time.Time(ts)
	default:
		return fmt.Errorf("expected timestamp, got %T", value)
	}
	return nil
}

func (a *timestampFieldAccessor) Delete(evt *beat.Event) error {
	return errDelTimestamp
}

func hasField(m common.MapStr, field string) bool {
	b, err := m.HasKey(field)
	return b && err != nil
}

func isMetadata(to string) bool {
	return to == "@metadata" || strings.HasPrefix(to, "@metadata.")
}
