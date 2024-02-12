// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/types/known/structpb"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	inputcursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/mito/lib"
)

type input struct {
	time func() time.Time
	cfg  config
}

const (
	inputName string = "websocket"
	root      string = "state"
)

func Plugin(log *logp.Logger, store inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "Websocket Input",
		Doc:        "Collect data from websocket api endpoints",
		Manager:    NewInputManager(log, store),
	}
}

func (input) Name() string { return inputName }

func (input) Test(src inputcursor.Source, _ v2.TestContext) error {
	return nil
}

// Run starts the input and blocks as long as websocket connections are alive. It will return on
// context cancellation or type invalidity errors, any other error will be retried.
func (input) Run(env v2.Context, src inputcursor.Source, crsr inputcursor.Cursor, pub inputcursor.Publisher) error {
	var cursor map[string]interface{}
	if !crsr.IsNew() { // Allow the user to bootstrap the program if needed.
		err := crsr.Unpack(&cursor)
		if err != nil {
			return err
		}
	}
	return input{}.run(env, src.(*source), cursor, pub)
}

func (i input) run(env v2.Context, src *source, cursor map[string]interface{}, pub inputcursor.Publisher) error {
	cfg := src.cfg
	i.cfg = cfg
	log := env.Logger.With("input_url", cfg.URL)

	metrics := newInputMetrics(env.ID)
	defer metrics.Close()
	metrics.url.Set(cfg.URL.String())
	metrics.errorsTotal.Set(0)

	ctx := ctxtool.FromCanceller(env.Cancelation)

	patterns, err := regexpsFromConfig(cfg)
	if err != nil {
		metrics.errorsTotal.Inc()
		return err
	}

	prg, ast, err := newProgram(ctx, cfg.Program, root, patterns, log)
	if err != nil {
		metrics.errorsTotal.Inc()
		return err
	}
	var state map[string]interface{}
	if cfg.State == nil {
		state = make(map[string]interface{})
	} else {
		state = cfg.State
	}
	if cursor != nil {
		state["cursor"] = cursor
	}

	// websocket client
	headers := formHeader(cfg)
	url := cfg.URL.String()
	c, resp, err := websocket.DefaultDialer.DialContext(ctx, url, headers)
	if resp != nil && resp.Body != nil {
		log.Debugw("websocket connection response", "body", resp.Body)
		resp.Body.Close()
	}
	if err != nil {
		metrics.errorsTotal.Inc()
		log.Errorw("failed to establish websocket connection", "error", err)
		return err
	}
	defer c.Close()

	done := make(chan error)

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				metrics.errorsTotal.Inc()
				if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Errorw("websocket connection closed", "error", err)
				} else {
					log.Errorw("failed to read websocket data", "error", err)
				}
				done <- err
				return
			}
			metrics.receivedBytesTotal.Add(uint64(len(message)))
			state["response"] = message
			log.Debugw("received websocket message", logp.Namespace("websocket"), string(message))
			err = i.processAndPublishData(ctx, metrics, prg, ast, state, cursor, pub, log)
			if err != nil {
				metrics.errorsTotal.Inc()
				log.Errorw("failed to process and publish data", "error", err)
				done <- err
				return
			}
		}
	}()

	// blocks until done is closed, context is cancelled or an error is received
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// processAndPublishData processes the data in state, updates the cursor and publishes it to the publisher.
// the CEL program here only executes a single time, since the websocket connection is persistent and events are received and processed in real time.
func (i *input) processAndPublishData(ctx context.Context, metrics *inputMetrics, prg cel.Program, ast *cel.Ast,
	state map[string]interface{}, cursor map[string]interface{}, pub inputcursor.Publisher, log *logp.Logger) error {
	goodCursor := cursor
	log.Debugw("cel engine state before eval", logp.Namespace("websocket"), "state", redactor{state: state, cfg: i.cfg.Redact})
	start := i.now().In(time.UTC)
	state, err := evalWith(ctx, prg, ast, state, start)
	log.Debugw("cel engine state after eval", logp.Namespace("websocket"), "state", redactor{state: state, cfg: i.cfg.Redact})
	if err != nil {
		metrics.celEvalErrors.Add(1)
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return err
		default:
			metrics.errorsTotal.Inc()
		}
		log.Errorw("failed evaluation", "error", err)
	}
	metrics.celProcessingTime.Update(time.Since(start).Nanoseconds())

	e, ok := state["events"]
	if !ok {
		log.Errorw("unexpected missing events from evaluation")
	}
	var events []interface{}
	switch e := e.(type) {
	case []interface{}:
		if len(e) == 0 {
			return nil
		}
		events = e
	case map[string]interface{}:
		if e == nil {
			return nil
		}
		log.Debugw("single event object returned by evaluation", "event", e)
		events = []interface{}{e}
	default:
		return fmt.Errorf("unexpected type returned for evaluation events: %T", e)
	}

	// We have a non-empty batch of events to process.
	metrics.batchesReceived.Add(1)
	metrics.eventsReceived.Add(uint64(len(events)))

	// Drop events from state. If we fail during the publication,
	// we will reprocess these events.
	delete(state, "events")

	// Get cursors if they exist.
	var (
		cursors      []interface{}
		singleCursor bool
	)
	if c, ok := state["cursor"]; ok {
		cursors, ok = c.([]interface{})
		if ok {
			if len(cursors) != len(events) {
				log.Errorw("unexpected cursor list length", "cursors", len(cursors), "events", len(events))
				// But try to continue.
				if len(cursors) < len(events) {
					cursors = nil
				}
			}
		} else {
			cursors = []interface{}{c}
			singleCursor = true
		}
	}
	// Drop old cursor from state. This will be replaced with
	// the current cursor object below; it is an array now.
	delete(state, "cursor")

	start = time.Now()
	var hadPublicationError bool
	for i, e := range events {
		event, ok := e.(map[string]interface{})
		if !ok {
			return fmt.Errorf("unexpected type returned for evaluation events: %T", e)
		}
		var pubCursor interface{}
		if cursors != nil {
			if singleCursor {
				// Only set the cursor for publication at the last event
				// when a single cursor object has been provided.
				if i == len(events)-1 {
					goodCursor = cursor
					cursor, ok = cursors[0].(map[string]interface{})
					if !ok {
						return fmt.Errorf("unexpected type returned for evaluation cursor element: %T", cursors[0])
					}
					pubCursor = cursor
				}
			} else {
				goodCursor = cursor
				cursor, ok = cursors[i].(map[string]interface{})
				if !ok {
					return fmt.Errorf("unexpected type returned for evaluation cursor element: %T", cursors[i])
				}
				pubCursor = cursor
			}
		}
		// Publish the event.
		err = pub.Publish(beat.Event{
			Timestamp: time.Now(),
			Fields:    event,
		}, pubCursor)
		if err != nil {
			hadPublicationError = true
			metrics.errorsTotal.Inc()
			log.Errorw("error publishing event", "error", err)
			cursors = nil // We are lost, so retry with this event's cursor,
			continue      // but continue with the events that we have without
			// advancing the cursor. This allows us to potentially publish the
			// events we have now, with a fallback to the last guaranteed
			// correctly published cursor.
		}
		if i == 0 {
			metrics.batchesPublished.Add(1)
		}
		metrics.eventsPublished.Add(1)

		err = ctx.Err()
		if err != nil {
			return err
		}
	}
	// calculate batch processing time
	metrics.batchProcessingTime.Update(time.Since(start).Nanoseconds())

	// Advance the cursor to the final state if there was no error during
	// publications. This is needed to transition to the next set of events.
	if !hadPublicationError {
		goodCursor = cursor
	}

	// Replace the last known good cursor.
	state["cursor"] = goodCursor

	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		metrics.errorsTotal.Inc()
		log.Infof("input stopped because context was cancelled with: %v", err)
		err = nil
	}
	return err
}

func evalWith(ctx context.Context, prg cel.Program, ast *cel.Ast, state map[string]interface{}, now time.Time) (map[string]interface{}, error) {
	out, _, err := prg.ContextEval(ctx, map[string]interface{}{
		// Replace global program "now" with current time. This is necessary
		// as the lib.Time now global is static at program instantiation time
		// which will persist over multiple evaluations. The lib.Time behaviour
		// is correct for mito where CEL program instances live for only a
		// single evaluation. Rather than incurring the cost of creating a new
		// cel.Program for each evaluation, shadow lib.Time's now with a new
		// value for each eval. We retain the lib.Time now global for
		// compatibility between CEL programs developed in mito with programs
		// run in the input.
		"now": now,
		root:  state,
	})
	if err != nil {
		err = lib.DecoratedError{AST: ast, Err: err}
	}
	if e := ctx.Err(); e != nil {
		err = e
	}
	if err != nil {
		state["events"] = errorMessage(fmt.Sprintf("failed eval: %v", err))
		clearWantMore(state)
		return state, fmt.Errorf("failed eval: %w", err)
	}

	v, err := out.ConvertToNative(reflect.TypeOf((*structpb.Struct)(nil)))
	if err != nil {
		state["events"] = errorMessage(fmt.Sprintf("failed proto conversion: %v", err))
		clearWantMore(state)
		return state, fmt.Errorf("failed proto conversion: %w", err)
	}
	switch v := v.(type) {
	case *structpb.Struct:
		return v.AsMap(), nil
	default:
		// This should never happen.
		errMsg := fmt.Sprintf("unexpected native conversion type: %T", v)
		state["events"] = errorMessage(errMsg)
		clearWantMore(state)
		return state, errors.New(errMsg)
	}
}

// now is time.Now with a modifiable time source.
func (i input) now() time.Time {
	if i.time == nil {
		return time.Now()
	}
	return i.time()
}

// clearWantMore sets the state to not request additional work in a periodic evaluation.
// It leaves state intact if there is no "want_more" element, and sets the element to false
// if there is. This is necessary instead of just doing delete(state, "want_more") as
// client CEL code may expect the want_more field to be present.
func clearWantMore(state map[string]interface{}) {
	if _, ok := state["want_more"]; ok {
		state["want_more"] = false
	}
}

func errorMessage(msg string) map[string]interface{} {
	return map[string]interface{}{"error": map[string]interface{}{"message": msg}}
}

func formHeader(cfg config) map[string][]string {
	header := make(map[string][]string)
	switch {
	case cfg.Auth.CustomAuth != nil:
		header[cfg.Auth.CustomAuth.Header] = []string{cfg.Auth.CustomAuth.Value}
	case cfg.Auth.BearerToken != "":
		header["Authorization"] = []string{"Bearer " + cfg.Auth.BearerToken}
	case cfg.Auth.BasicToken != "":
		header["Authorization"] = []string{"Basic " + cfg.Auth.BasicToken}
	}
	return header
}
