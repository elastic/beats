// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package dispatcher

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/pipeline/actions"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage/store"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
)

type actionHandlers map[string]actions.Handler

// ActionDispatcher processes actions coming from fleet using registered set of handlers.
type ActionDispatcher struct {
	ctx      context.Context
	log      *logger.Logger
	handlers actionHandlers
	def      actions.Handler
}

// New creates a new action dispatcher.
func New(ctx context.Context, log *logger.Logger, def actions.Handler) (*ActionDispatcher, error) {
	var err error
	if log == nil {
		log, err = logger.New("action_dispatcher", false)
		if err != nil {
			return nil, err
		}
	}

	if def == nil {
		return nil, errors.New("missing default handler")
	}

	return &ActionDispatcher{
		ctx:      ctx,
		log:      log,
		handlers: make(actionHandlers),
		def:      def,
	}, nil
}

// Register registers a new handler for action.
func (ad *ActionDispatcher) Register(a fleetapi.Action, handler actions.Handler) error {
	k := ad.key(a)
	_, ok := ad.handlers[k]
	if ok {
		return fmt.Errorf("action with type %T is already registered", a)
	}
	ad.handlers[k] = handler
	return nil
}

// MustRegister registers a new handler for action.
// Panics if not successful.
func (ad *ActionDispatcher) MustRegister(a fleetapi.Action, handler actions.Handler) {
	err := ad.Register(a, handler)
	if err != nil {
		panic("could not register action, error: " + err.Error())
	}
}

func (ad *ActionDispatcher) key(a fleetapi.Action) string {
	return reflect.TypeOf(a).String()
}

// Dispatch dispatches an action using pre-registered set of handlers.
func (ad *ActionDispatcher) Dispatch(ctx context.Context, acker store.FleetAcker, actions ...fleetapi.Action) (err error) {
	// span, ctx := apm.StartSpan(ctx, "dispatch", "app.internal")
	// defer func() {
	// 	if err != nil {
	// 		apm.CaptureError(ctx, err).Send()
	// 	}
	// 	span.End()
	// }()

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
		if err = ad.ctx.Err(); err != nil {
			return err
		}

		if err = ad.dispatchAction(action, acker); err != nil {
			ad.log.Debugf("Failed to dispatch action '%+v', error: %+v", action, err)
			return err
		}
		ad.log.Debugf("Successfully dispatched action: '%+v'", action)
	}

	err = acker.Commit(ctx)
	return err
}

func (ad *ActionDispatcher) dispatchAction(a fleetapi.Action, acker store.FleetAcker) error {
	handler, found := ad.handlers[(ad.key(a))]
	if !found {
		return ad.def.Handle(ad.ctx, a, acker)
	}

	return handler.Handle(ad.ctx, a, acker)
}

func detectTypes(actions []fleetapi.Action) []string {
	str := make([]string, len(actions))
	for idx, action := range actions {
		str[idx] = reflect.TypeOf(action).String()
	}
	return str
}
