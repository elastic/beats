// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
)

type action = fleetapi.Action

type actionHandler interface {
	Handle(ctx context.Context, a action, acker fleetAcker) error
}

type actionHandlers map[string]actionHandler

type actionDispatcher struct {
	ctx      context.Context
	log      *logger.Logger
	handlers actionHandlers
	def      actionHandler
}

func newActionDispatcher(ctx context.Context, log *logger.Logger, def actionHandler) (*actionDispatcher, error) {
	var err error
	if log == nil {
		log, err = logger.New()
		if err != nil {
			return nil, err
		}
	}

	if def == nil {
		return nil, errors.New("missing default handler")
	}

	return &actionDispatcher{
		ctx:      ctx,
		log:      log,
		handlers: make(actionHandlers),
		def:      def,
	}, nil
}

func (ad *actionDispatcher) Register(a action, handler actionHandler) error {
	k := ad.key(a)
	_, ok := ad.handlers[k]
	if ok {
		return fmt.Errorf("action with type %T is already registered", a)
	}
	ad.handlers[k] = handler
	return nil
}

func (ad *actionDispatcher) MustRegister(a action, handler actionHandler) {
	err := ad.Register(a, handler)
	if err != nil {
		panic("could not register action, error: " + err.Error())
	}
}

func (ad *actionDispatcher) key(a action) string {
	return reflect.TypeOf(a).String()
}

func (ad *actionDispatcher) Dispatch(acker fleetAcker, actions ...action) error {
	if len(actions) == 0 {
		ad.log.Debug("No action to dispatch")
		return nil
	}

	ad.log.Debugf(
		"Dispatch %d actions of types: %s",
		len(actions),
		strings.Join(detectTypes(actions), ", "),
	)

	for _, action := range actions {
		if err := ad.dispatchAction(action, acker); err != nil {
			ad.log.Debugf("Failed to dispatch action '%+v', error: %+v", action, err)
			return err
		}
		ad.log.Debugf("Successfully dispatched action: '%+v'", action)
	}

	return acker.Commit(ad.ctx)
}

func (ad *actionDispatcher) dispatchAction(a action, acker fleetAcker) error {
	handler, found := ad.handlers[(ad.key(a))]
	if !found {
		return ad.def.Handle(ad.ctx, a, acker)
	}

	return handler.Handle(ad.ctx, a, acker)
}

func detectTypes(actions []action) []string {
	str := make([]string, len(actions))
	for idx, action := range actions {
		str[idx] = reflect.TypeOf(action).String()
	}
	return str
}
