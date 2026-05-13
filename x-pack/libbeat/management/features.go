// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

import (
	"fmt"

	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
	conf "github.com/elastic/elastic-agent-libs/config"
)

// NewConfigFromProto converts the given *proto.Features object to
// a *config.C object.
//
// Explicitly not defined in libbeat/features as proto.Features is ELv2 licensed.
func NewConfigFromProto(f *proto.Features) (*conf.C, error) {
	if f == nil {
		return nil, nil
	}

	var beatCfg struct {
		Features *proto.Features `config:"features"`
	}

	beatCfg.Features = f

	c, err := conf.NewConfigFrom(&beatCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to parse feature flags message into beat configuration: %w", err)
	}

	_, err = c.Remove("features.source", -1)
	if err != nil {
		return nil, fmt.Errorf("unable to convert feature flags message to beat configuration: %w", err)
	}

	return c, nil
}
