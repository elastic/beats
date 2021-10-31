// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package wasm

import (
	"bytes"
	"context"
	"sync"

	"github.com/bytecodealliance/wasmtime-go"

	"github.com/open-policy-agent/opa/internal/wasm/sdk/opa/errors"
	"github.com/open-policy-agent/opa/internal/wasm/util"
	"github.com/open-policy-agent/opa/metrics"
)

var errNotReady = errors.New(errors.NotReadyErr, "")

// Pool maintains a pool of WebAssemly VM instances.
type Pool struct {
	engine         *wasmtime.Engine
	available      chan struct{}
	mutex          sync.Mutex
	dataMtx        sync.Mutex
	initialized    bool
	closed         bool
	policy         []byte
	parsedData     []byte // Parsed parsedData memory segment, used to seed new VM's
	parsedDataAddr int32  // Address for parsedData value root, used to seed new VM's
	memoryMinPages uint32
	memoryMaxPages uint32
	vms            []*VM // All current VM instances, acquired or not.
	acquired       []bool
	pendingReinit  *VM
	blockedReinit  chan struct{}
}

// NewPool constructs a new pool with the pool and VM configuration provided.
func NewPool(poolSize, memoryMinPages, memoryMaxPages uint32) *Pool {

	cfg := wasmtime.NewConfig()
	cfg.SetInterruptable(true)

	available := make(chan struct{}, poolSize)
	for i := uint32(0); i < poolSize; i++ {
		available <- struct{}{}
	}

	return &Pool{
		engine:         wasmtime.NewEngineWithConfig(cfg),
		memoryMinPages: memoryMinPages,
		memoryMaxPages: memoryMaxPages,
		available:      available,
		vms:            make([]*VM, 0),
		acquired:       make([]bool, 0),
	}
}

// ParsedData returns a reference to the pools parsed external data used to
// initialize new VM's.
func (p *Pool) ParsedData() (int32, []byte) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.parsedDataAddr, p.parsedData
}

// Policy returns the raw policy Wasm module used by VM's in the pool
func (p *Pool) Policy() []byte {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.policy
}

// Size returns the current number of VM's in the pool
func (p *Pool) Size() int {
	return len(p.vms)
}

// Acquire obtains a VM from the pool, waiting if all VMms are in use
// and building one as necessary. Returns either ErrNotReady or
// ErrInternal if an error.
func (p *Pool) Acquire(ctx context.Context, metrics metrics.Metrics) (*VM, error) {
	metrics.Timer("wasm_pool_acquire").Start()
	defer metrics.Timer("wasm_pool_acquire").Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-p.available:
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.initialized || p.closed {
		return nil, errNotReady
	}

	for i, vm := range p.vms {
		if !p.acquired[i] {
			p.acquired[i] = true
			return vm, nil
		}
	}

	policy, parsedData, parsedDataAddr := p.policy, p.parsedData, p.parsedDataAddr

	p.mutex.Unlock()
	vm, err := newVM(vmOpts{
		policy:         policy,
		data:           nil,
		parsedData:     parsedData,
		parsedDataAddr: parsedDataAddr,
		memoryMin:      p.memoryMinPages,
		memoryMax:      p.memoryMaxPages,
	}, p.engine)
	p.mutex.Lock()

	if err != nil {
		p.available <- struct{}{}
		return nil, errors.New(errors.InternalErr, err.Error())
	}

	p.acquired = append(p.acquired, true)
	p.vms = append(p.vms, vm)
	return vm, nil
}

// Release releases the VM back to the pool.
func (p *Pool) Release(vm *VM, metrics metrics.Metrics) {
	metrics.Timer("wasm_pool_release").Start()
	defer metrics.Timer("wasm_pool_release").Stop()

	p.mutex.Lock()

	// If the policy data setting is waiting for this one, don't release it back to the general consumption.
	// Note the reinit is responsible for pushing to available channel once done with the VM.
	if vm == p.pendingReinit {
		p.mutex.Unlock()
		p.blockedReinit <- struct{}{}
		return
	}

	for i := range p.vms {
		if p.vms[i] == vm {
			p.acquired[i] = false
			p.mutex.Unlock()
			p.available <- struct{}{}
			return
		}
	}

	// VM instance not found anymore, hence pool reconfigured and can release the VM.

	p.mutex.Unlock()
	p.available <- struct{}{}
}

// SetPolicyData re-initializes the vms within the pool with the new policy
// and data. The re-initialization takes place atomically: all new vms
// are constructed in advance before touching the pool.  Returns
// either ErrNotReady, ErrInvalidPolicy or ErrInternal if an error
// occurs.
func (p *Pool) SetPolicyData(ctx context.Context, policy []byte, data []byte) error {
	p.dataMtx.Lock()
	defer p.dataMtx.Unlock()

	p.mutex.Lock()

	if !p.initialized {
		vm, err := newVM(vmOpts{
			policy:         policy,
			data:           data,
			parsedData:     nil,
			parsedDataAddr: 0,
			memoryMin:      p.memoryMinPages,
			memoryMax:      p.memoryMaxPages,
		}, p.engine)

		if err == nil {
			parsedDataAddr, parsedData := vm.cloneDataSegment()
			p.memoryMinPages = util.Pages(uint32(vm.memory.DataSize(vm.store)))
			p.vms = append(p.vms, vm)
			p.acquired = append(p.acquired, false)
			p.initialized = true
			p.policy, p.parsedData, p.parsedDataAddr = policy, parsedData, parsedDataAddr
		} else {
			err = errors.New(errors.InvalidPolicyOrDataErr, err.Error())
		}

		p.mutex.Unlock()
		return err
	}

	if p.closed {
		p.mutex.Unlock()
		return errNotReady
	}

	currentPolicy, currentData := p.policy, p.parsedData
	p.mutex.Unlock()

	if bytes.Equal(policy, currentPolicy) && bytes.Equal(data, currentData) {
		return nil
	}

	err := p.setPolicyData(ctx, policy, data)
	if err != nil {
		return errors.New(errors.InternalErr, err.Error())
	}

	return nil
}

// SetDataPath will update the current data on the VMs by setting the value at the
// specified path. If an error occurs the instance is still in a valid state, however
// the data will not have been modified.
func (p *Pool) SetDataPath(ctx context.Context, path []string, value interface{}) error {
	p.dataMtx.Lock()
	defer p.dataMtx.Unlock()
	return p.updateVMs(func(vm *VM, opts vmOpts) error {
		return vm.SetDataPath(ctx, path, value)
	})
}

// RemoveDataPath will update the current data on the VMs by removing the value at the
// specified path. If an error occurs the instance is still in a valid state, however
// the data will not have been modified.
func (p *Pool) RemoveDataPath(ctx context.Context, path []string) error {
	p.dataMtx.Lock()
	defer p.dataMtx.Unlock()
	return p.updateVMs(func(vm *VM, _ vmOpts) error {
		return vm.RemoveDataPath(ctx, path)
	})
}

// setPolicyData reinitializes the VMs one at a time.
func (p *Pool) setPolicyData(ctx context.Context, policy []byte, data []byte) error {
	return p.updateVMs(func(vm *VM, opts vmOpts) error {
		opts.policy = policy
		opts.data = data
		return vm.SetPolicyData(ctx, opts)
	})
}

// updateVMs Iterates over each VM, waiting for each to safely acquire them,
// and applies the update function. If the first update succeeds any subsequent
// failures will remove the VM and continue through the pool. Otherwise an error
// will be returned.
func (p *Pool) updateVMs(update func(vm *VM, opts vmOpts) error) error {
	var policy []byte
	var parsedData []byte
	var parsedDataAddr int32
	seedMemorySize := p.memoryMinPages
	activated := false
	i := 0
	for {
		vm := p.Wait(i)
		if vm == nil {
			// All have been updated or removed.
			return nil
		}

		err := update(vm, vmOpts{
			policy:         policy,
			parsedData:     parsedData,
			parsedDataAddr: parsedDataAddr,
			memoryMin:      seedMemorySize,
			memoryMax:      p.memoryMaxPages, // The max pages cannot be changed while updating.
		})

		if err != nil {
			// No guarantee about the VM state after an error; hence, remove.
			p.remove(i)
			p.Release(vm, metrics.New())

			// After the first successful activation, proceed through all the VMs, ignoring the remaining errors.
			if !activated {
				return err
			}
			// Note: Do not increment i when it has been removed! That index is
			// replaced by the last VM in the list so we must re-run with the
			// same index.
		} else {
			if !activated {
				// Activate the policy and data, now that a single VM has been reset without errors.
				activated = true
				policy = vm.policy
				parsedDataAddr, parsedData = vm.cloneDataSegment()
				seedMemorySize = util.Pages(uint32(vm.memory.DataSize(vm.store)))
				p.activate(policy, parsedData, parsedDataAddr, seedMemorySize)
			}

			p.Release(vm, metrics.New())

			// Only increment on success
			i++
		}
	}
}

// Close waits for all the evaluations to finish and then releases the VMs.
func (p *Pool) Close() {
	for range p.vms {
		<-p.available
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.closed = true
	p.vms = nil
}

// Wait steals the i'th VM instance. The VM has to be released afterwards.
func (p *Pool) Wait(i int) *VM {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if i == len(p.vms) {
		return nil
	}

	vm := p.vms[i]
	isActive := p.acquired[i]
	p.acquired[i] = true

	if isActive {
		p.blockedReinit = make(chan struct{}, 1)
		p.pendingReinit = vm
	}

	p.mutex.Unlock()

	if isActive {
		<-p.blockedReinit
	} else {
		<-p.available
	}

	p.mutex.Lock()
	p.pendingReinit = nil
	return vm
}

// remove removes the i'th vm.
func (p *Pool) remove(i int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	n := len(p.vms)
	if n > 1 {
		p.vms[i] = p.vms[n-1]
		p.acquired[i] = p.acquired[n-1]
	}

	p.vms = p.vms[0 : n-1]
	p.acquired = p.acquired[0 : n-1]
}

func (p *Pool) activate(policy []byte, data []byte, dataAddr int32, minMemoryPages uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.policy, p.parsedData, p.parsedDataAddr, p.memoryMinPages = policy, data, dataAddr, minMemoryPages
}
