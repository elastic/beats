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

//go:build windows

package eventlog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	win "github.com/elastic/beats/v7/winlogbeat/sys/wineventlog"
)

// fakeRenderer only supports a full render, like XMLRenderer.
type fakeRenderer struct {
	event   winevent.Event
	renders int
}

func (r *fakeRenderer) Render(win.EvtHandle) (*winevent.Event, string, error) {
	r.renders++
	evt := r.event
	return &evt, "<xml/>", nil
}

func (r *fakeRenderer) Close() error { return nil }

// fakeSystemRenderer can render system properties separately, like Renderer.
type fakeSystemRenderer struct {
	fakeRenderer
	systemRenders    int
	remainingRenders int
}

func (r *fakeSystemRenderer) RenderSystem(win.EvtHandle) (*winevent.Event, error) {
	r.systemRenders++
	evt := r.event
	return &evt, nil
}

func (r *fakeSystemRenderer) RenderRemaining(_ win.EvtHandle, evt *winevent.Event) (*winevent.Event, string, error) {
	r.remainingRenders++
	return evt, "", nil
}

func TestRenderFiltersBeforeFullRender(t *testing.T) {
	filter, err := newRecordFilter(query{EventID: "100"})
	require.NoError(t, err, "filter must build")

	unwanted := winevent.Event{EventIdentifier: winevent.EventIdentifier{ID: 200}}

	t.Run("system renderer skips remaining render for filtered events", func(t *testing.T) {
		r := &fakeSystemRenderer{fakeRenderer: fakeRenderer{event: unwanted}}
		l := &winEventLog{renderer: r, filter: filter}

		evt, _, filtered, err := l.render(0)
		require.NoError(t, err, "render must not fail")
		assert.True(t, filtered, "event should be filtered")
		assert.NotNil(t, evt, "system properties must be returned for checkpointing")
		assert.Equal(t, 1, r.systemRenders, "system properties should be rendered")
		assert.Equal(t, 0, r.remainingRenders, "remaining render should be skipped")
		assert.Equal(t, 0, r.renders, "full render should not be used")
	})

	t.Run("plain renderer filters after full render", func(t *testing.T) {
		r := &fakeRenderer{event: unwanted}
		l := &winEventLog{renderer: r, filter: filter}

		evt, _, filtered, err := l.render(0)
		require.NoError(t, err, "render must not fail")
		assert.True(t, filtered, "event should be filtered")
		assert.NotNil(t, evt, "event must be returned for checkpointing")
		assert.Equal(t, 1, r.renders, "full render should be used")
	})
}
