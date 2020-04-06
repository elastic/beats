// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/reload"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/x-pack/libbeat/management/api"
)

type reloadable struct {
	reloaded chan *reload.ConfigWithMeta
}

func (r *reloadable) Reload(c *reload.ConfigWithMeta) error {
	r.reloaded <- c
	return nil
}

func TestConfigManager(t *testing.T) {
	registry := reload.NewRegistry()
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating id: %v", err)
	}
	accessToken := "footoken"
	reloadable := reloadable{
		reloaded: make(chan *reload.ConfigWithMeta, 1),
	}
	registry.MustRegister("test.block", &reloadable)

	mux := http.NewServeMux()
	i := 0
	responses := []string{
		// Initial load
		`{"success": true, "list":[{"type":"test.block","config":{"module":"apache2"}}]}`,

		// No change, no reload
		`{"success": true, "list":[{"type":"test.block","config":{"module":"apache2"}}]}`,

		// Changed, reload
		`{"success": true, "list":[{"type":"test.block","config":{"module":"system"}}]}`,
	}
	mux.Handle(fmt.Sprintf("/api/beats/agent/%s/configuration", id), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, responses[i])
		i++
	}))

	reporter := addEventsReporterHandle(mux, id)

	server := httptest.NewServer(mux)

	c, err := api.ConfigFromURL(server.URL, common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Enabled:     true,
		Mode:        ModeCentralManagement,
		Period:      100 * time.Millisecond,
		Kibana:      c,
		AccessToken: accessToken,
		EventsReporter: EventReporterConfig{
			Period:       50 * time.Millisecond,
			MaxBatchSize: 1,
		},
	}

	manager, err := NewConfigManagerWithConfig(config, registry, id)
	if err != nil {
		t.Fatal(err)
	}

	manager.Start()

	// On first reload we will get apache2 module
	config1 := <-reloadable.reloaded
	assert.Equal(t, &reload.ConfigWithMeta{
		Config: common.MustNewConfigFrom(map[string]interface{}{
			"module": "apache2",
		}),
	}, config1)

	config2 := <-reloadable.reloaded
	assert.Equal(t, &reload.ConfigWithMeta{
		Config: common.MustNewConfigFrom(map[string]interface{}{
			"module": "system",
		}),
	}, config2)

	// Cleanup
	manager.Stop()
	os.Remove(paths.Resolve(paths.Data, "management.yml"))

	events := []api.Event{&Starting, &InProgress, &Running, &InProgress, &Running, &Stopped}
	assertEvents(t, events, reporter)
}

func TestRemoveItems(t *testing.T) {
	registry := reload.NewRegistry()
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating id: %v", err)
	}
	accessToken := "footoken"
	reloadable := reloadable{
		reloaded: make(chan *reload.ConfigWithMeta, 1),
	}
	registry.MustRegister("test.blocks", &reloadable)

	mux := http.NewServeMux()
	i := 0
	responses := []string{
		// Initial load
		`{"success": true, "list":[{"type":"test.blocks","config":{"module":"apache2"}}]}`,

		// Return no blocks
		`{"success": true, "list":[]}`,
	}
	mux.Handle(fmt.Sprintf("/api/beats/agent/%s/configuration", id), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, responses[i])
		i++
	}))

	reporter := addEventsReporterHandle(mux, id)
	server := httptest.NewServer(mux)

	c, err := api.ConfigFromURL(server.URL, common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Enabled:     true,
		Mode:        ModeCentralManagement,
		Period:      100 * time.Millisecond,
		Kibana:      c,
		AccessToken: accessToken,
		EventsReporter: EventReporterConfig{
			Period:       50 * time.Millisecond,
			MaxBatchSize: 1,
		},
	}

	manager, err := NewConfigManagerWithConfig(config, registry, id)
	if err != nil {
		t.Fatal(err)
	}

	manager.Start()

	// On first reload we will get apache2 module
	config1 := <-reloadable.reloaded
	assert.Equal(t, &reload.ConfigWithMeta{
		Config: common.MustNewConfigFrom(map[string]interface{}{
			"module": "apache2",
		}),
	}, config1)

	// Get a nil config, even if the block is not part of the payload
	config2 := <-reloadable.reloaded
	var nilConfig *reload.ConfigWithMeta
	assert.Equal(t, nilConfig, config2)

	// Cleanup
	manager.Stop()
	os.Remove(paths.Resolve(paths.Data, "management.yml"))

	events := []api.Event{&Starting, &InProgress, &Running, &InProgress, &Running, &Stopped}
	assertEvents(t, events, reporter)
}

func responseText(s string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, s)
	}
}

func TestUnEnroll(t *testing.T) {
	registry := reload.NewRegistry()
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating id: %v", err)
	}
	accessToken := "footoken"
	reloadable := reloadable{
		reloaded: make(chan *reload.ConfigWithMeta, 1),
	}
	registry.MustRegister("test.blocks", &reloadable)

	mux := http.NewServeMux()
	i := 0
	responses := []http.HandlerFunc{ // Initial load
		responseText(`{"success": true, "list":[{"type":"test.blocks","config":{"module":"apache2"}}]}`),
		http.NotFound,
	}

	mux.Handle(fmt.Sprintf("/api/beats/agent/%s/configuration", id), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responses[i](w, r)
		i++
	}))

	reporter := addEventsReporterHandle(mux, id)
	server := httptest.NewServer(mux)

	c, err := api.ConfigFromURL(server.URL, common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Enabled:     true,
		Mode:        ModeCentralManagement,
		Period:      100 * time.Millisecond,
		Kibana:      c,
		AccessToken: accessToken,
		EventsReporter: EventReporterConfig{
			Period:       50 * time.Millisecond,
			MaxBatchSize: 1,
		},
	}

	manager, err := NewConfigManagerWithConfig(config, registry, id)
	if err != nil {
		t.Fatal(err)
	}

	manager.Start()

	// On first reload we will get apache2 module
	config1 := <-reloadable.reloaded
	assert.Equal(t, &reload.ConfigWithMeta{
		Config: common.MustNewConfigFrom(map[string]interface{}{
			"module": "apache2",
		}),
	}, config1)

	// Get a nil config, even if the block is not part of the payload
	config2 := <-reloadable.reloaded
	var nilConfig *reload.ConfigWithMeta
	assert.Equal(t, nilConfig, config2)

	// Cleanup
	manager.Stop()
	os.Remove(paths.Resolve(paths.Data, "management.yml"))

	events := []api.Event{&Starting, &InProgress, &Running, &InProgress, &Running, &Stopped}
	assertEvents(t, events, reporter)
}

func TestBadConfig(t *testing.T) {
	registry := reload.NewRegistry()
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating id: %v", err)
	}
	accessToken := "footoken"
	reloadable := reloadable{
		reloaded: make(chan *reload.ConfigWithMeta, 1),
	}
	registry.MustRegister("test.blocks", &reloadable)

	mux := http.NewServeMux()
	i := 0
	responses := []http.HandlerFunc{ // Initial load
		responseText(`{"success": true, "list":[{"type":"output","config":{"_sub_type": "console", "path": "/tmp/bad"}}]}`),
		// will not resend new events
		responseText(`{"success": true, "list":[{"type":"output","config":{"_sub_type": "console", "path": "/tmp/bad"}}]}`),
		// recover on call
		http.NotFound,
	}

	mux.Handle(fmt.Sprintf("/api/beats/agent/%s/configuration", id), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responses[i](w, r)
		i++
	}))

	reporter := addEventsReporterHandle(mux, id)
	server := httptest.NewServer(mux)

	c, err := api.ConfigFromURL(server.URL, common.NewConfig())
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Enabled:     true,
		Mode:        ModeCentralManagement,
		Period:      100 * time.Millisecond,
		Kibana:      c,
		AccessToken: accessToken,
		EventsReporter: EventReporterConfig{
			Period:       50 * time.Millisecond,
			MaxBatchSize: 1,
		},
		Blacklist: ConfigBlacklistSettings{
			Patterns: map[string]string{
				"output": "console|file",
			},
		},
	}

	manager, err := NewConfigManagerWithConfig(config, registry, id)
	if err != nil {
		t.Fatal(err)
	}

	manager.Start()

	// On first reload we will get apache2 module
	config1 := <-reloadable.reloaded
	assert.Nil(t, config1)

	// Cleanup
	manager.Stop()
	os.Remove(paths.Resolve(paths.Data, "management.yml"))

	events := []api.Event{
		&Starting,
		&InProgress,
		&Error{Type: ConfigError, Err: errors.New("Config for 'output' is blacklisted")},
		&Failed,
		&InProgress, // recovering on NotFound, to get out of the blocking.
		&Running,
		&Stopped,
	}
	assertEvents(t, events, reporter)
}

type testEventRequest struct {
	EventType api.EventType
	Event     api.Event
}

func (er *testEventRequest) UnmarshalJSON(b []byte) error {
	resp := struct {
		EventType api.EventType   `json:"type"`
		Event     json.RawMessage `json:"event"`
	}{}

	if err := json.Unmarshal(b, &resp); err != nil {
		return err
	}

	switch resp.EventType {
	case ErrorEvent:
		event := &Error{}
		if err := json.Unmarshal(resp.Event, event); err != nil {
			return err
		}
		*er = testEventRequest{EventType: resp.EventType, Event: event}
		return nil
	case StateEvent:
		event := State("")
		if err := json.Unmarshal(resp.Event, &event); err != nil {
			return err
		}
		*er = testEventRequest{EventType: resp.EventType, Event: &event}
		return nil
	}
	return fmt.Errorf("unknown event type of '%s'", resp.EventType)
}

// collect in the background any events generated from the managers.
type collectEventRequest struct {
	sync.Mutex
	requests []testEventRequest
}

func (r *collectEventRequest) Requests() []testEventRequest {
	r.Lock()
	defer r.Unlock()
	return r.requests
}

func (r *collectEventRequest) Add(requests ...testEventRequest) {
	r.Lock()
	defer r.Unlock()
	r.requests = append(r.requests, requests...)
}

func addEventsReporterHandle(mux *http.ServeMux, uuid uuid.UUID) *collectEventRequest {
	reporter := &collectEventRequest{}
	path := "/api/beats/" + uuid.String() + "/events"
	fn := func(w http.ResponseWriter, r *http.Request) {
		var requests []testEventRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&requests); err != nil {
			http.Error(w, "could not decode JSON", 500)
		}

		reporter.Add(requests...)
		resp := api.EventAPIResponse{Response: make([]api.EventResponse, len(requests))}

		for i := 0; i < len(requests); i++ {
			resp.Response[i] = api.EventResponse{BaseResponse: api.BaseResponse{Success: true}}
		}

		json.NewEncoder(w).Encode(resp)
	}
	mux.Handle(path, http.HandlerFunc(fn))
	return reporter
}

func assertEvents(t *testing.T, events []api.Event, reporter *collectEventRequest) {
	requests := reporter.Requests()
	if !assert.Equal(t, len(events), len(requests)) {
		return
	}

	for i := 0; i < len(events); i++ {
		switch v := requests[i].Event.(type) {
		case *State:
			assert.Equal(t, events[i], requests[i].Event)
		case *Error:
			comparable := events[i].(*Error)
			assert.Error(t, comparable.Err, v.Err)
		default:
			t.Fatalf("cannot assert unknown type: %T", requests[i].Event)
		}
	}
}

func TestConfigValidate(t *testing.T) {
	tests := map[string]struct {
		config *common.Config
		err    bool
	}{
		"missing access_token": {
			config: common.MustNewConfigFrom(map[string]interface{}{}),
			err:    true,
		},
		"access_token is present": {
			config: common.MustNewConfigFrom(map[string]interface{}{"access_token": "abc1234"}),
			err:    false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			c := defaultConfig()
			err := test.config.Unpack(c)
			if assert.NoError(t, err) {
				return
			}

			err = validateConfig(c)
			if test.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
