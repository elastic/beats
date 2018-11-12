// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/reload"
	"github.com/elastic/beats/x-pack/libbeat/management/api"
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
		`{"configuration_blocks":[{"type":"test.block","config":{"module":"apache2"}}]}`,

		// No change, no reload
		`{"configuration_blocks":[{"type":"test.block","config":{"module":"apache2"}}]}`,

		// Changed, reload
		`{"configuration_blocks":[{"type":"test.block","config":{"module":"system"}}]}`,
	}
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/api/beats/agent/%s/configuration?validSetting=true", id), r.RequestURI)
		fmt.Fprintf(w, responses[i])
		i++
	}))

	server := httptest.NewServer(mux)

	c, err := api.ConfigFromURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Enabled:     true,
		Period:      100 * time.Millisecond,
		Kibana:      c,
		AccessToken: accessToken,
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
		`{"configuration_blocks":[{"type":"test.blocks","config":{"module":"apache2"}}]}`,

		// Return no blocks
		`{"configuration_blocks":[]}`,
	}
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, responses[i])
		i++
	}))

	server := httptest.NewServer(mux)

	c, err := api.ConfigFromURL(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Enabled:     true,
		Period:      100 * time.Millisecond,
		Kibana:      c,
		AccessToken: accessToken,
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
}
