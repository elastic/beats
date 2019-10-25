// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/program"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/sorted"
)

// defautlRK default routing keys until we implement the routing key / config matrix.
var defautlRK = "DEFAULT"

type routingKey = string

type pipeline interface {
	Execute(*configRequest) error
	Close() error
}

type pipelineFunc func(*logger.Logger, routingKey) (pipeline, error)

type router struct {
	log             *logger.Logger
	routes          *sorted.Set
	pipelineFactory pipelineFunc
}

func newRouter(log *logger.Logger, factory pipelineFunc) (*router, error) {
	var err error
	if log == nil {
		log, err = logger.New()
		if err != nil {
			return nil, err
		}
	}
	return &router{log: log, pipelineFactory: factory, routes: sorted.NewSet()}, nil
}

func (r *router) Dispatch(id string, grpProg map[routingKey][]program.Program) error {
	s := sorted.NewSet()

	// Make sure that starting and updating is always done in the same order.
	for rk, programs := range grpProg {
		s.Add(rk, programs)
	}

	active := make(map[string]bool, len(grpProg))
	for _, rk := range s.Keys() {
		active[rk] = true

		// Are we already runnings this pipeline?
		// If we don't we create it otherwise we just forward the config request.
		p, ok := r.routes.Get(rk)
		var err error
		if !ok {
			r.log.Debugf("Creating pipeline %s", rk)
			p, err = r.pipelineFactory(r.log, rk)
			if err != nil {
				return err
			}
			r.routes.Add(rk, p)
		}

		programs, ok := s.Get(rk)
		if !ok {
			return fmt.Errorf("could not find programs for routing key %s", rk)
		}

		req := &configRequest{
			id:       id,
			programs: programs.([]program.Program),
		}

		r.log.Debugf(
			"Pipeline %s need to run config with ID %s and programs: %s",
			rk,
			req.ShortID(),
			strings.Join(req.ProgramNames(), ", "),
		)

		err = p.(pipeline).Execute(req)
		if err != nil {
			return err
		}
	}

	// cleanup inactive pipelines.
	// pipelines are shutdown down in alphabetical order.
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

		p.(pipeline).Close()
		r.routes.Remove(k)
	}

	return nil
}
