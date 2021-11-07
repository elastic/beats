// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package opa

import (
	"context"
	"encoding/json"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/open-policy-agent/opa/internal/wasm/sdk/internal/wasm"
	"github.com/open-policy-agent/opa/internal/wasm/sdk/opa/errors"
	sdk_errors "github.com/open-policy-agent/opa/internal/wasm/sdk/opa/errors"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/topdown/cache"
	"github.com/open-policy-agent/opa/topdown/print"
)

var errNotReady = errors.New(errors.NotReadyErr, "")

// OPA executes WebAssembly compiled Rego policies.
type OPA struct {
	configErr      error // Delayed configuration error, if any.
	memoryMinPages uint32
	memoryMaxPages uint32 // 0 means no limit.
	poolSize       uint32
	pool           *wasm.Pool
	mutex          sync.Mutex // To serialize access to SetPolicy, SetData and Close.
	policy         []byte     // Current policy.
	data           []byte     // Current data.
	logError       func(error)
}

// Result holds the evaluation result.
type Result struct {
	Result []byte
}

// New constructs a new OPA SDK instance, ready to be configured with
// With functions. If no policy is provided as a part of
// configuration, policy (and data) needs to be set before invoking
// Eval. Once constructed and configured, the instance needs to be
// initialized before invoking the Eval.
func New() *OPA {
	opa := &OPA{
		memoryMinPages: 16,
		memoryMaxPages: 0x10000, // 4GB
		poolSize:       uint32(runtime.GOMAXPROCS(0)),
		logError:       func(error) {},
	}

	return opa
}

// Init initializes the SDK instance after the construction and
// configuration. If the configuration is invalid, it returns
// ErrInvalidConfig.
func (o *OPA) Init() (*OPA, error) {
	ctx := context.Background()
	if o.configErr != nil {
		return nil, o.configErr
	}

	o.pool = wasm.NewPool(o.poolSize, o.memoryMinPages, o.memoryMaxPages)

	if len(o.policy) != 0 {
		if err := o.pool.SetPolicyData(ctx, o.policy, o.data); err != nil {
			return nil, err
		}
	}

	return o, nil
}

// SetData updates the data for the subsequent Eval calls.  Returns
// either ErrNotReady, ErrInvalidPolicyOrData, or ErrInternal if an
// error occurs.
func (o *OPA) SetData(ctx context.Context, v interface{}) error {
	if o.pool == nil {
		return errNotReady
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return sdk_errors.New(sdk_errors.InvalidPolicyOrDataErr, err.Error())
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.setPolicyData(ctx, o.policy, raw)
}

// SetDataPath will update the current data on the VMs by setting the value at the
// specified path. If an error occurs the instance is still in a valid state, however
// the data will not have been modified.
func (o *OPA) SetDataPath(ctx context.Context, path []string, value interface{}) error {
	return o.pool.SetDataPath(ctx, path, value)
}

// RemoveDataPath will update the current data on the VMs by removing the value at the
// specified path. If an error occurs the instance is still in a valid state, however
// the data will not have been modified.
func (o *OPA) RemoveDataPath(ctx context.Context, path []string) error {
	return o.pool.RemoveDataPath(ctx, path)
}

// SetPolicy updates the policy for the subsequent Eval calls.
// Returns either ErrNotReady, ErrInvalidPolicy or ErrInternal if an
// error occurs.
func (o *OPA) SetPolicy(ctx context.Context, p []byte) error {
	if o.pool == nil {
		return errNotReady
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.setPolicyData(ctx, p, o.data)
}

// SetPolicyData updates both the policy and data for the subsequent
// Eval calls.  Returns either ErrNotReady, ErrInvalidPolicyOrData, or
// ErrInternal if an error occurs.
func (o *OPA) SetPolicyData(ctx context.Context, policy []byte, data *interface{}) error {
	if o.pool == nil {
		return errNotReady
	}

	var raw []byte
	if data != nil {
		var err error
		raw, err = json.Marshal(*data)
		if err != nil {
			return sdk_errors.New(sdk_errors.InvalidPolicyOrDataErr, err.Error())
		}
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	return o.setPolicyData(ctx, policy, raw)
}

func (o *OPA) setPolicyData(ctx context.Context, policy []byte, data []byte) error {
	if err := o.pool.SetPolicyData(ctx, policy, data); err != nil {
		return err
	}

	o.policy = policy
	o.data = data
	return nil
}

// EvalOpts define options for performing an evaluation
type EvalOpts struct {
	Entrypoint             int32
	Input                  *interface{}
	Metrics                metrics.Metrics
	Time                   time.Time
	Seed                   io.Reader
	InterQueryBuiltinCache cache.InterQueryCache
	PrintHook              print.Hook
}

// Eval evaluates the policy with the given input, returning the
// evaluation results. If no policy was configured at construction
// time nor set after, the function returns ErrNotReady.  It returns
// ErrInternal if any other error occurs.
func (o *OPA) Eval(ctx context.Context, opts EvalOpts) (*Result, error) {
	if o.pool == nil {
		return nil, errNotReady
	}

	m := opts.Metrics
	if m == nil {
		m = metrics.New()
	}

	instance, err := o.pool.Acquire(ctx, m)
	if err != nil {
		return nil, err
	}

	defer o.pool.Release(instance, m)

	result, err := instance.Eval(ctx, opts.Entrypoint, opts.Input, m, opts.Seed, opts.Time, opts.InterQueryBuiltinCache, opts.PrintHook)
	if err != nil {
		return nil, err
	}

	return &Result{Result: result}, nil
}

// Close waits until all the pending evaluations complete and then
// releases all the resources allocated. Eval will return ErrClosed
// afterwards.
func (o *OPA) Close() {
	if o.pool == nil {
		return
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	o.pool.Close()
}

// Entrypoints returns a mapping of entrypoint name to ID for use by Eval() and EvalBool().
func (o *OPA) Entrypoints(ctx context.Context) (map[string]int32, error) {
	instance, err := o.pool.Acquire(ctx, metrics.New())
	if err != nil {
		return nil, err
	}

	defer o.pool.Release(instance, metrics.New())

	return instance.Entrypoints(), nil
}
