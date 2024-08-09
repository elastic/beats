// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
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
	stream StreamFollower

	time func() time.Time
	cfg  config
}

type StreamFollower interface {
	FollowStream(context.Context) error
	Close() error
}

const (
	inputName string = "streaming"
	root      string = "state"
)

func Plugin(log *logp.Logger, store inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "Streaming Input",
		Doc:        "Collect data from streaming data sources",
		Manager:    NewInputManager(log, store),
	}
}

func PluginWebsocketAlias(log *logp.Logger, store inputcursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       "websocket",
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "Websocket Input",
		Doc:        "Collect data from websocket data sources",
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

func (i input) run(env v2.Context, src *source, cursor map[string]any, pub inputcursor.Publisher) error {
	cfg := src.cfg
	log := env.Logger.With("input_url", cfg.URL)

	ctx := ctxtool.FromCanceller(env.Cancelation)
	s, err := NewWebsocketFollower(ctx, env.ID, cfg, cursor, pub, log, i.time)
	if err != nil {
		return err
	}
	defer s.Close()

	return s.FollowStream(ctx)
}

// getURL initializes the input URL with the help of the url_program.
func getURL(ctx context.Context, name, src, url string, state map[string]any, redaction *redact, log *logp.Logger, now func() time.Time) (string, error) {
	if src == "" {
		return url, nil
	}

	state["url"] = url
	// CEL program which is used to prime/initialize the input url
	url_prg, ast, err := newProgram(ctx, src, root, nil, log)
	if err != nil {
		return "", err
	}

	log.Debugw("cel engine state before url_eval", logp.Namespace("websocket"), "state", redactor{state: state, cfg: redaction})
	start := now().In(time.UTC)
	url, err = evalURLWith(ctx, url_prg, ast, state, start)
	log.Debugw("url_eval result", logp.Namespace(name), "modified_url", url)
	if err != nil {
		log.Errorw("failed url evaluation", "error", err)
		return "", err
	}
	return url, nil
}

func evalURLWith(ctx context.Context, prg cel.Program, ast *cel.Ast, state map[string]interface{}, now time.Time) (string, error) {
	out, err := evalRefVal(ctx, prg, ast, state, now)
	if err != nil {
		return "", fmt.Errorf("failed eval: %w", err)
	}
	v, err := out.ConvertToNative(reflect.TypeOf(""))
	if err != nil {
		return "", fmt.Errorf("failed type conversion: %w", err)
	}
	switch v := v.(type) {
	case string:
		_, err = url.Parse(v)
		return v, err
	default:
		// This should never happen.
		return "", fmt.Errorf("unexpected native conversion type: %T", v)
	}
}

// processor is a common CEL program evaluator and event publisher.
type processor struct {
	prg cel.Program
	ast *cel.Ast
	pub inputcursor.Publisher

	ns      string
	log     *logp.Logger
	redact  *redact
	metrics *inputMetrics
}

// process processes the data in state, updates the cursor and publishes it to
// the reciever's publisher. The CEL program here only executes a single time,
// since the connection is persistent and events are received and processed in
// real time.
func (p processor) process(ctx context.Context, state, cursor map[string]any, start time.Time) error {
	goodCursor := cursor
	p.log.Debugw("cel engine state before eval", logp.Namespace(p.ns), "state", redactor{state: state, cfg: p.redact})
	state, err := evalWith(ctx, p.prg, p.ast, state, start)
	p.log.Debugw("cel engine state after eval", logp.Namespace(p.ns), "state", redactor{state: state, cfg: p.redact})
	if err != nil {
		p.metrics.celEvalErrors.Add(1)
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return err
		default:
			p.metrics.errorsTotal.Inc()
		}
		p.log.Errorw("failed evaluation", "error", err)
	}
	p.metrics.celProcessingTime.Update(time.Since(start).Nanoseconds())

	e, ok := state["events"]
	if !ok {
		p.log.Errorw("unexpected missing events from evaluation")
	}
	var events []any
	switch e := e.(type) {
	case []any:
		if len(e) == 0 {
			return nil
		}
		events = e
	case map[string]any:
		if e == nil {
			return nil
		}
		p.log.Debugw("single event object returned by evaluation", "event", e)
		events = []any{e}
	default:
		return fmt.Errorf("unexpected type returned for evaluation events: %T", e)
	}

	// We have a non-empty batch of events to process.
	p.metrics.batchesReceived.Add(1)
	p.metrics.eventsReceived.Add(uint64(len(events)))

	// Drop events from state. If we fail during the publication,
	// we will reprocess these events.
	delete(state, "events")

	// Get cursors if they exist.
	var (
		cursors      []any
		singleCursor bool
	)
	if c, ok := state["cursor"]; ok {
		cursors, ok = c.([]any)
		if ok {
			if len(cursors) != len(events) {
				p.log.Errorw("unexpected cursor list length", "cursors", len(cursors), "events", len(events))
				// But try to continue.
				if len(cursors) < len(events) {
					cursors = nil
				}
			}
		} else {
			cursors = []any{c}
			singleCursor = true
		}
	}
	// Drop old cursor from state. This will be replaced with
	// the current cursor object below; it is an array now.
	delete(state, "cursor")

	start = time.Now()
	var hadPublicationError bool
	for i, e := range events {
		event, ok := e.(map[string]any)
		if !ok {
			return fmt.Errorf("unexpected type returned for evaluation events: %T", e)
		}
		var pubCursor any
		if cursors != nil {
			if singleCursor {
				// Only set the cursor for publication at the last event
				// when a single cursor object has been provided.
				if i == len(events)-1 {
					goodCursor = cursor
					cursor, ok = cursors[0].(map[string]any)
					if !ok {
						return fmt.Errorf("unexpected type returned for evaluation cursor element: %T", cursors[0])
					}
					pubCursor = cursor
				}
			} else {
				goodCursor = cursor
				cursor, ok = cursors[i].(map[string]any)
				if !ok {
					return fmt.Errorf("unexpected type returned for evaluation cursor element: %T", cursors[i])
				}
				pubCursor = cursor
			}
		}
		// Publish the event.
		err = p.pub.Publish(beat.Event{
			Timestamp: time.Now(),
			Fields:    event,
		}, pubCursor)
		if err != nil {
			hadPublicationError = true
			p.metrics.errorsTotal.Inc()
			p.log.Errorw("error publishing event", "error", err)
			cursors = nil // We are lost, so retry with this event's cursor,
			continue      // but continue with the events that we have without
			// advancing the cursor. This allows us to potentially publish the
			// events we have now, with a fallback to the last guaranteed
			// correctly published cursor.
		}
		if i == 0 {
			p.metrics.batchesPublished.Add(1)
		}
		p.metrics.eventsPublished.Add(1)

		err = ctx.Err()
		if err != nil {
			return err
		}
	}
	// calculate batch processing time
	p.metrics.batchProcessingTime.Update(time.Since(start).Nanoseconds())

	// Advance the cursor to the final state if there was no error during
	// publications. This is needed to transition to the next set of events.
	if !hadPublicationError {
		goodCursor = cursor
	}

	// Replace the last known good cursor.
	state["cursor"] = goodCursor

	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		p.metrics.errorsTotal.Inc()
		p.log.Infof("input stopped because context was cancelled with: %v", err)
		err = nil
	}
	return err
}

func evalWith(ctx context.Context, prg cel.Program, ast *cel.Ast, state map[string]interface{}, now time.Time) (map[string]interface{}, error) {
	out, err := evalRefVal(ctx, prg, ast, state, now)
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

func evalRefVal(ctx context.Context, prg cel.Program, ast *cel.Ast, state map[string]interface{}, now time.Time) (ref.Val, error) {
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
	return out, err
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
