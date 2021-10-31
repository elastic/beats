// Copyright 2021 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

// Import this package to enable evaluation of rego code using the
// built-in wasm engine.
package wasm

import (
	"context"

	"github.com/open-policy-agent/opa/internal/rego/opa"
	wopa "github.com/open-policy-agent/opa/internal/wasm/sdk/opa"
)

func init() {
	opa.RegisterEngine("wasm", &factory{})
}

// OPA is an implementation of the OPA SDK.
type OPA struct {
	opa *wopa.OPA
}

type factory struct{}

// New constructs a new OPA instance.
func (*factory) New() opa.EvalEngine {
	return &OPA{opa: wopa.New()}
}

// WithPolicyBytes configures the compiled policy to load.
func (o *OPA) WithPolicyBytes(policy []byte) opa.EvalEngine {
	o.opa = o.opa.WithPolicyBytes(policy)
	return o
}

// WithDataJSON configures the JSON data to load.
func (o *OPA) WithDataJSON(data interface{}) opa.EvalEngine {
	o.opa = o.opa.WithDataJSON(data)
	return o
}

// Init initializes the OPA instance.
func (o *OPA) Init() (opa.EvalEngine, error) {
	i, err := o.opa.Init()
	if err != nil {
		return nil, err
	}
	o.opa = i
	return o, nil
}

func (o *OPA) Entrypoints(ctx context.Context) (map[string]int32, error) {
	return o.opa.Entrypoints(ctx)
}

// Eval evaluates the policy.
func (o *OPA) Eval(ctx context.Context, opts opa.EvalOpts) (*opa.Result, error) {
	evalOptions := wopa.EvalOpts{
		Input:                  opts.Input,
		Metrics:                opts.Metrics,
		Entrypoint:             opts.Entrypoint,
		Time:                   opts.Time,
		Seed:                   opts.Seed,
		InterQueryBuiltinCache: opts.InterQueryBuiltinCache,
		PrintHook:              opts.PrintHook,
	}

	res, err := o.opa.Eval(ctx, evalOptions)
	if err != nil {
		return nil, err
	}

	return &opa.Result{Result: res.Result}, nil
}

func (o *OPA) SetData(ctx context.Context, data interface{}) error {
	return o.opa.SetData(ctx, data)
}

func (o *OPA) SetDataPath(ctx context.Context, path []string, data interface{}) error {
	return o.opa.SetDataPath(ctx, path, data)
}

func (o *OPA) RemoveDataPath(ctx context.Context, path []string) error {
	return o.opa.RemoveDataPath(ctx, path)
}

func (o *OPA) Close() {
	o.opa.Close()
}
