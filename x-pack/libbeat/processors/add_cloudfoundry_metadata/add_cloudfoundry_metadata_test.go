// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux darwin windows

package add_cloudfoundry_metadata

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func TestNoClient(t *testing.T) {
	p := addCloudFoundryMetadata{}

	evt := beat.Event{}
	observed, err := p.Run(&evt)
	assert.NoError(t, err)
	assert.Equal(t, evt, *observed)
}

func TestNoCFApp(t *testing.T) {
	p := addCloudFoundryMetadata{
		client: &fakeClient{},
	}

	evt := beat.Event{
		Fields: common.MapStr{
			"cloudfoundry": common.MapStr{
				"app": common.MapStr{},
			},
		},
	}
	observed, err := p.Run(&evt)
	assert.NoError(t, err)
	assert.Equal(t, evt, *observed)
}

func TestCFAppIdInvalid(t *testing.T) {
	p := addCloudFoundryMetadata{
		client: &fakeClient{},
	}

	evt := beat.Event{
		Fields: common.MapStr{
			"cloudfoundry": common.MapStr{
				"app": common.MapStr{
					"id": 1,
				},
			},
		},
	}
	observed, err := p.Run(&evt)
	assert.NoError(t, err)
	assert.Equal(t, evt, *observed)
}

func TestCFAppNotFound(t *testing.T) {
	p := addCloudFoundryMetadata{
		log:    logp.NewLogger("add_cloudfoundry_metadata"),
		client: &fakeClient{},
	}

	evt := beat.Event{
		Fields: common.MapStr{
			"cloudfoundry": common.MapStr{
				"app": common.MapStr{
					"id": mustCreateFakeGuid(),
				},
			},
		},
	}
	observed, err := p.Run(&evt)
	assert.NoError(t, err)
	assert.Equal(t, evt, *observed)
}

func TestCFAppUpdated(t *testing.T) {
	guid := mustCreateFakeGuid()
	app := cfclient.App{
		Guid: guid,
		Name: "My Fake App",
		SpaceData: cfclient.SpaceResource{
			Meta: cfclient.Meta{
				Guid: mustCreateFakeGuid(),
			},
			Entity: cfclient.Space{
				Name: "My Fake Space",
				OrgData: cfclient.OrgResource{
					Meta: cfclient.Meta{
						Guid: mustCreateFakeGuid(),
					},
					Entity: cfclient.Org{
						Name: "My Fake Org",
					},
				},
			},
		},
	}
	p := addCloudFoundryMetadata{
		log:    logp.NewLogger("add_cloudfoundry_metadata"),
		client: &fakeClient{app},
	}

	evt := beat.Event{
		Fields: common.MapStr{
			"cloudfoundry": common.MapStr{
				"app": common.MapStr{
					"id": guid,
				},
			},
		},
	}
	expected := beat.Event{
		Fields: common.MapStr{
			"cloudfoundry": common.MapStr{
				"app": common.MapStr{
					"id":   guid,
					"name": app.Name,
				},
				"space": common.MapStr{
					"id":   app.SpaceData.Meta.Guid,
					"name": app.SpaceData.Entity.Name,
				},
				"org": common.MapStr{
					"id":   app.SpaceData.Entity.OrgData.Meta.Guid,
					"name": app.SpaceData.Entity.OrgData.Entity.Name,
				},
			},
		},
	}
	observed, err := p.Run(&evt)
	assert.NoError(t, err)
	assert.Equal(t, expected, *observed)
}

type fakeClient struct {
	app cfclient.App
}

func (c *fakeClient) GetAppByGuid(guid string) (*cfclient.App, error) {
	if c.app.Guid != guid {
		return nil, fmt.Errorf("unknown app")
	}
	return &c.app, nil
}

func (c *fakeClient) StartJanitor(_ time.Duration) {
}

func (c *fakeClient) StopJanitor() {
}

func mustCreateFakeGuid() string {
	uuid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return uuid.String()
}
