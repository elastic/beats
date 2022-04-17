// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package add_cloudfoundry_metadata

import (
	"testing"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
	"github.com/menderesk/beats/v7/x-pack/libbeat/common/cloudfoundry"
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

func TestCFAppMetadataAlreadyPresent(t *testing.T) {
	guid := mustCreateFakeGuid()
	app := cloudfoundry.AppMeta{
		Guid:      guid,
		Name:      "My Fake App",
		SpaceGuid: mustCreateFakeGuid(),
		SpaceName: "My Fake Space",
		OrgGuid:   mustCreateFakeGuid(),
		OrgName:   "My Fake Org",
	}
	p := addCloudFoundryMetadata{
		log:    logp.NewLogger("add_cloudfoundry_metadata"),
		client: &fakeClient{app},
	}

	evt := beat.Event{
		Fields: common.MapStr{
			"cloudfoundry": common.MapStr{
				"app": common.MapStr{
					"id":   guid,
					"name": "Other App Name",
				},
				"space": common.MapStr{
					"id":   app.SpaceGuid,
					"name": app.SpaceName,
				},
				"org": common.MapStr{
					"id":   app.OrgGuid,
					"name": app.OrgName,
				},
			},
		},
	}
	expected := beat.Event{
		Fields: common.MapStr{
			"cloudfoundry": common.MapStr{
				"app": common.MapStr{
					"id":   guid,
					"name": "Other App Name",
				},
				"space": common.MapStr{
					"id":   app.SpaceGuid,
					"name": app.SpaceName,
				},
				"org": common.MapStr{
					"id":   app.OrgGuid,
					"name": app.OrgName,
				},
			},
		},
	}
	observed, err := p.Run(&evt)
	assert.NoError(t, err)
	assert.Equal(t, expected, *observed)
}

func TestCFAppUpdated(t *testing.T) {
	guid := mustCreateFakeGuid()
	app := cloudfoundry.AppMeta{
		Guid:      guid,
		Name:      "My Fake App",
		SpaceGuid: mustCreateFakeGuid(),
		SpaceName: "My Fake Space",
		OrgGuid:   mustCreateFakeGuid(),
		OrgName:   "My Fake Org",
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
					"id":   app.SpaceGuid,
					"name": app.SpaceName,
				},
				"org": common.MapStr{
					"id":   app.OrgGuid,
					"name": app.OrgName,
				},
			},
		},
	}
	observed, err := p.Run(&evt)
	assert.NoError(t, err)
	assert.Equal(t, expected, *observed)
}

type fakeClient struct {
	app cloudfoundry.AppMeta
}

func (c *fakeClient) GetAppByGuid(guid string) (*cloudfoundry.AppMeta, error) {
	if c.app.Guid != guid {
		return nil, cfclient.CloudFoundryError{Code: 100004}
	}
	return &c.app, nil
}

func (c *fakeClient) Close() error {
	return nil
}

func mustCreateFakeGuid() string {
	uuid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	return uuid.String()
}
