// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package v2

import (
	"sync"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/go-concert/chorus"
)

type ConfiguredInput struct {
	Info  string
	Input RunnableInput
}

type RunnableInput interface {
	Test(*chorus.Closer, *logp.Logger) error
	Run(Context) error
}

type simpleRunner struct {
	info    string
	context Context
	input   RunnableInput

	mu           sync.Mutex
	activeCloser *chorus.Closer
	waiter       sync.WaitGroup
}

func (i *ConfiguredInput) TestInput(closer *chorus.Closer, log *logp.Logger) error {
	return i.Input.Test(closer, log)
}

func (i *ConfiguredInput) CreateRunner(ctx Context) (Runner, error) {
	return &simpleRunner{info: i.Info, context: ctx, input: i.Input}, nil
}

func (r *simpleRunner) String() string {
	return r.info
}

func (r *simpleRunner) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.activeCloser != nil {
		return
	}

	r.activeCloser = chorus.WithCloser(r.context.Closer, nil)
	r.waiter.Add(1)
	go func() {
		defer r.waiter.Done()
		r.run()
	}()
}

func (r *simpleRunner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.activeCloser == nil {
		return
	}

	r.activeCloser.Close()
	r.activeCloser = nil
	r.waiter.Wait()
}

func (r *simpleRunner) run() {
	// setup managed pipeline to close any connections an input has kept open
	// on return.  The pipeline ensure that the context.Closer is installed if
	// no custom Closer has been configured on connect.
	managedPipeline := &managedPipeline{
		pipeline: r.context.Pipeline,
		closer:   r.activeCloser,
	}
	defer managedPipeline.Close()

	// setup managed store accessor, that keeps track of all stores being
	// used by the input.
	// Still open stores are automatically deactivated if run returns (prevent
	// possible resource leaks).
	managedStore := &managedStoreAccessor{accessor: r.context.StoreAccessor}
	defer managedStore.shutdown()

	ctx := r.context
	ctx.Closer = r.activeCloser
	ctx.Pipeline = managedPipeline
	ctx.StoreAccessor = managedStore

	// TODO: capture panic and report it

	var err error
	if ctx.Status != nil {
		ctx.Status.Starting()
		defer reportDone(&err, ctx.Status)
	}
	err = r.input.Run(ctx)
}

func reportDone(err *error, reporter StatusObserver) {
	if *err == nil {
		ctx.Status.Stopped()
	} else {
		ctx.Status.Failed(*err)
	}
}
