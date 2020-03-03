// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/program"
)

const shortID = 8

type configRequest struct {
	id        string
	createdAt time.Time
	programs  []program.Program
}

func (c *configRequest) String() string {
	names := c.ProgramNames()
	return "[" + c.ShortID() + "] Config: " + strings.Join(names, ", ")
}

func (c *configRequest) ID() string {
	return c.id
}

func (c *configRequest) ShortID() string {
	if len(c.id) <= shortID {
		return c.id
	}
	return c.id[0:shortID]
}

func (c *configRequest) CreatedAt() time.Time {
	return c.createdAt
}

func (c *configRequest) Programs() []program.Program {
	return c.programs
}

func (c *configRequest) ProgramNames() []string {
	names := make([]string, 0, len(c.programs))
	for _, name := range c.programs {
		names = append(names, name.Spec.Name)
	}
	return names
}
