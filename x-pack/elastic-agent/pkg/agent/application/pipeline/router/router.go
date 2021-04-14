// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package router

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/sorted"
)

type router struct {
	log           *logger.Logger
	routes        *sorted.Set
	streamFactory pipeline.StreamFunc
}

// New creates a new router.
func New(log *logger.Logger, factory pipeline.StreamFunc) (pipeline.Router, error) {
	var err error
	if log == nil {
		log, err = logger.New("router", false)
		if err != nil {
			return nil, err
		}
	}
	return &router{log: log, streamFactory: factory, routes: sorted.NewSet()}, nil
}

func (r *router) Routes() *sorted.Set {
	return r.routes
}

func (r *router) Route(id string, grpProg map[pipeline.RoutingKey][]program.Program) error {
	s := sorted.NewSet()

	// Make sure that starting and updating is always done in the same order.
	for rk, programs := range grpProg {
		s.Add(rk, programs)
	}

	active := make(map[string]bool, len(grpProg))
	for _, rk := range s.Keys() {
		active[rk] = true

		// Are we already runnings this streams?
		// When it doesn't exist we just create it, if it already exist we forward the configuration.
		p, ok := r.routes.Get(rk)
		var err error
		if !ok {
			r.log.Debugf("Creating stream: %s", rk)
			p, err = r.streamFactory(r.log, rk)
			if err != nil {
				return err
			}
			r.routes.Add(rk, p)
		}

		programs, ok := s.Get(rk)
		if !ok {
			return fmt.Errorf("could not find programs for routing key %s", rk)
		}

		req := configrequest.New(id, time.Now(), programs.([]program.Program))

		r.log.Debugf(
			"Streams %s need to run config with ID %s and programs: %s",
			rk,
			req.ShortID(),
			strings.Join(req.ProgramNames(), ", "),
		)

		err = p.(pipeline.Stream).Execute(req)
		if err != nil {
			return err
		}
	}

	// cleanup inactive streams.
	// streams are shutdown down in alphabetical order.
	keys := r.routes.Keys()
	for _, k := range keys {
		_, ok := active[k]
		if ok {
			continue
		}

		p, ok := r.routes.Get(k)
		if !ok {
			continue
		}

		r.log.Debugf("Removing routing key %s", k)

		p.(pipeline.Stream).Close()
		r.routes.Remove(k)
	}

	return nil
}

// Shutdown shutdowns the router because Agent is stopping.
func (r *router) Shutdown() {
	keys := r.routes.Keys()
	for _, k := range keys {
		p, ok := r.routes.Get(k)
		if !ok {
			continue
		}
		p.(pipeline.Stream).Shutdown()
		r.routes.Remove(k)
	}
}
