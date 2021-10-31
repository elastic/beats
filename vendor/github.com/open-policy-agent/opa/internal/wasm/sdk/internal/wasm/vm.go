// Copyright 2020 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package wasm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/bytecodealliance/wasmtime-go"

	"github.com/open-policy-agent/opa/ast"
	sdk_errors "github.com/open-policy-agent/opa/internal/wasm/sdk/opa/errors"
	"github.com/open-policy-agent/opa/internal/wasm/util"
	"github.com/open-policy-agent/opa/metrics"
	"github.com/open-policy-agent/opa/topdown"
	"github.com/open-policy-agent/opa/topdown/cache"
	"github.com/open-policy-agent/opa/topdown/print"
)

// VM is a wrapper around a Wasm VM instance
type VM struct {
	dispatcher           *builtinDispatcher
	engine               *wasmtime.Engine
	store                *wasmtime.Store
	instance             *wasmtime.Instance // Pointer to avoid unintented destruction (triggering finalizers within).
	intHandle            *wasmtime.InterruptHandle
	policy               []byte
	abiMajorVersion      int32
	abiMinorVersion      int32
	memory               *wasmtime.Memory
	memoryMin            uint32
	memoryMax            uint32
	entrypointIDs        map[string]int32
	baseHeapPtr          int32
	dataAddr             int32
	evalHeapPtr          int32
	evalOneOff           func(context.Context, int32, int32, int32, int32, int32) (int32, error)
	eval                 func(context.Context, int32) error
	evalCtxGetResult     func(context.Context, int32) (int32, error)
	evalCtxNew           func(context.Context) (int32, error)
	evalCtxSetData       func(context.Context, int32, int32) error
	evalCtxSetInput      func(context.Context, int32, int32) error
	evalCtxSetEntrypoint func(context.Context, int32, int32) error
	heapPtrGet           func(context.Context) (int32, error)
	heapPtrSet           func(context.Context, int32) error
	jsonDump             func(context.Context, int32) (int32, error)
	jsonParse            func(context.Context, int32, int32) (int32, error)
	valueDump            func(context.Context, int32) (int32, error)
	valueParse           func(context.Context, int32, int32) (int32, error)
	malloc               func(context.Context, int32) (int32, error)
	free                 func(context.Context, int32) error
	valueAddPath         func(context.Context, int32, int32, int32) (int32, error)
	valueRemovePath      func(context.Context, int32, int32) (int32, error)
}

type vmOpts struct {
	policy         []byte
	data           []byte
	parsedData     []byte
	parsedDataAddr int32
	memoryMin      uint32
	memoryMax      uint32
}

func newVM(opts vmOpts, engine *wasmtime.Engine) (*VM, error) {
	ctx := context.Background()
	v := &VM{engine: engine}
	store := wasmtime.NewStore(engine)
	memorytype := wasmtime.NewMemoryType(opts.memoryMin, true, opts.memoryMax)
	memory, err := wasmtime.NewMemory(store, memorytype)
	if err != nil {
		return nil, err
	}

	module, err := wasmtime.NewModule(store.Engine, opts.policy)
	if err != nil {
		return nil, err
	}

	linker := wasmtime.NewLinker(store.Engine)
	v.dispatcher = newBuiltinDispatcher()
	externs := opaFunctions(v.dispatcher, store)
	for name, extern := range externs {
		if err := linker.Define("env", name, extern); err != nil {
			return nil, fmt.Errorf("linker: env.%s: %w", name, err)
		}
	}
	if err := linker.Define("env", "memory", memory); err != nil {
		return nil, fmt.Errorf("linker: env.memory: %w", err)
	}

	i, err := linker.Instantiate(store, module)
	if err != nil {
		return nil, err
	}
	v.intHandle, err = store.InterruptHandle()
	if err != nil {
		return nil, fmt.Errorf("get interrupt handle: %w", err)
	}

	v.abiMajorVersion, v.abiMinorVersion, err = getABIVersion(i, store)
	if err != nil {
		return nil, fmt.Errorf("invalid module: %w", err)
	}
	if v.abiMajorVersion != int32(1) || (v.abiMinorVersion != int32(1) && v.abiMinorVersion != int32(2)) {
		return nil, fmt.Errorf("invalid module: unsupported ABI version: %d.%d", v.abiMajorVersion, v.abiMinorVersion)
	}

	// re-exported import, or just plain export if memory wasn't imported
	memory = i.GetExport(store, "memory").Memory()

	v.store = store
	v.instance = i
	v.policy = opts.policy
	v.memory = memory
	v.memoryMin = opts.memoryMin
	v.memoryMax = opts.memoryMax
	v.entrypointIDs = make(map[string]int32)
	v.dataAddr = 0
	v.eval = func(ctx context.Context, a int32) error { return callVoid(ctx, v, "eval", a) }
	v.evalCtxGetResult = func(ctx context.Context, a int32) (int32, error) { return call(ctx, v, "opa_eval_ctx_get_result", a) }
	v.evalCtxNew = func(ctx context.Context) (int32, error) { return call(ctx, v, "opa_eval_ctx_new") }
	v.evalCtxSetData = func(ctx context.Context, a int32, b int32) error {
		return callVoid(ctx, v, "opa_eval_ctx_set_data", a, b)
	}
	v.evalCtxSetInput = func(ctx context.Context, a int32, b int32) error {
		return callVoid(ctx, v, "opa_eval_ctx_set_input", a, b)
	}
	v.evalOneOff = func(ctx context.Context, ep, dataAddr, inputAddr, inputLen, heapAddr int32) (int32, error) {
		return call(ctx, v, "opa_eval", 0 /* reserved */, ep, dataAddr, inputAddr, inputLen, heapAddr, 1 /* value output */)
	}
	v.evalCtxSetEntrypoint = func(ctx context.Context, a int32, b int32) error {
		return callVoid(ctx, v, "opa_eval_ctx_set_entrypoint", a, b)
	}
	v.free = func(ctx context.Context, a int32) error { return callVoid(ctx, v, "opa_free", a) }
	v.heapPtrGet = func(ctx context.Context) (int32, error) { return call(ctx, v, "opa_heap_ptr_get") }
	v.heapPtrSet = func(ctx context.Context, a int32) error { return callVoid(ctx, v, "opa_heap_ptr_set", a) }
	v.jsonDump = func(ctx context.Context, a int32) (int32, error) { return call(ctx, v, "opa_json_dump", a) }
	v.jsonParse = func(ctx context.Context, a int32, b int32) (int32, error) {
		return call(ctx, v, "opa_json_parse", a, b)
	}
	v.valueDump = func(ctx context.Context, a int32) (int32, error) { return call(ctx, v, "opa_value_dump", a) }
	v.valueParse = func(ctx context.Context, a int32, b int32) (int32, error) {
		return call(ctx, v, "opa_value_parse", a, b)
	}
	v.malloc = func(ctx context.Context, a int32) (int32, error) { return call(ctx, v, "opa_malloc", a) }
	v.valueAddPath = func(ctx context.Context, a int32, b int32, c int32) (int32, error) {
		return call(ctx, v, "opa_value_add_path", a, b, c)
	}
	v.valueRemovePath = func(ctx context.Context, a int32, b int32) (int32, error) {
		return call(ctx, v, "opa_value_remove_path", a, b)
	}

	// Initialize the heap.

	if _, err := v.malloc(ctx, 0); err != nil {
		return nil, err
	}

	if v.baseHeapPtr, err = v.getHeapState(ctx); err != nil {
		return nil, err
	}

	// Optimization for cloning a vm, if provided a parsed data memory buffer
	// insert it directly into the new vm's buffer and set pointers accordingly.
	// This only works because the placement is deterministic (eg, for a given policy
	// the base heap pointer and parsed data layout will always be the same).
	if opts.parsedData != nil {
		if uint32(memory.DataSize(store))-uint32(v.baseHeapPtr) < uint32(len(opts.parsedData)) {
			delta := uint32(len(opts.parsedData)) - (uint32(memory.DataSize(store)) - uint32(v.baseHeapPtr))
			_, err = memory.Grow(store, uint64(util.Pages(delta)))
			if err != nil {
				return nil, err
			}
		}
		mem := memory.UnsafeData(store)
		for src, dest := 0, v.baseHeapPtr; src < len(opts.parsedData); src, dest = src+1, dest+1 {
			mem[dest] = opts.parsedData[src]
		}
		v.dataAddr = opts.parsedDataAddr
		v.evalHeapPtr = v.baseHeapPtr + int32(len(opts.parsedData))
		err := v.setHeapState(ctx, v.evalHeapPtr)
		if err != nil {
			return nil, err
		}
	} else if opts.data != nil {
		if v.dataAddr, err = v.toRegoJSON(ctx, opts.data, true); err != nil {
			return nil, err
		}
	}

	if v.evalHeapPtr, err = v.getHeapState(ctx); err != nil {
		return nil, err
	}

	// Construct the builtin id to name mappings.

	val, err := i.GetFunc(store, "builtins").Call(store)
	if err != nil {
		return nil, err
	}

	builtins, err := v.fromRegoJSON(ctx, val.(int32), true)
	if err != nil {
		return nil, err
	}

	builtinMap := map[int32]topdown.BuiltinFunc{}

	for name, id := range builtins.(map[string]interface{}) {
		f := topdown.GetBuiltin(name)
		if f == nil {
			return nil, fmt.Errorf("builtin '%s' not found", name)
		}

		n, err := id.(json.Number).Int64()
		if err != nil {
			panic(err)
		}

		builtinMap[int32(n)] = f
	}

	v.dispatcher.SetMap(builtinMap)

	// Extract the entrypoint ID's
	val, err = i.GetFunc(store, "entrypoints").Call(store)
	if err != nil {
		return nil, err
	}

	epMap, err := v.fromRegoJSON(ctx, val.(int32), true)
	if err != nil {
		return nil, err
	}

	for ep, value := range epMap.(map[string]interface{}) {
		id, err := value.(json.Number).Int64()
		if err != nil {
			return nil, err
		}
		v.entrypointIDs[ep] = int32(id)
	}

	return v, nil
}

func getABIVersion(i *wasmtime.Instance, store wasmtime.Storelike) (int32, int32, error) {
	major := i.GetExport(store, "opa_wasm_abi_version").Global()
	minor := i.GetExport(store, "opa_wasm_abi_minor_version").Global()
	if major != nil && minor != nil {
		majorVal := major.Get(store)
		minorVal := minor.Get(store)
		if majorVal.Kind() == wasmtime.KindI32 && minorVal.Kind() == wasmtime.KindI32 {
			return majorVal.I32(), minorVal.I32(), nil
		}
	}
	return 0, 0, fmt.Errorf("failed to read ABI version")
}

// Eval performs an evaluation of the specified entrypoint, with any provided
// input, and returns the resulting value dumped to a string.
func (i *VM) Eval(ctx context.Context,
	entrypoint int32,
	input *interface{},
	metrics metrics.Metrics,
	seed io.Reader,
	ns time.Time,
	iqbCache cache.InterQueryCache,
	ph print.Hook) ([]byte, error) {
	if i.abiMinorVersion < int32(2) {
		return i.evalCompat(ctx, entrypoint, input, metrics, seed, ns, iqbCache, ph)
	}

	metrics.Timer("wasm_vm_eval").Start()
	defer metrics.Timer("wasm_vm_eval").Stop()

	mem := i.memory.UnsafeData(i.store)
	inputAddr, inputLen := int32(0), int32(0)

	// NOTE: we'll never free the memory used for the input string during
	// the one evaluation, but we'll overwrite it on the next evaluation.
	heapPtr := i.evalHeapPtr

	if input != nil {
		metrics.Timer("wasm_vm_eval_prepare_input").Start()
		var raw []byte
		switch v := (*input).(type) {
		case []byte:
			raw = v
		case *ast.Term:
			raw = []byte(v.String())
		case ast.Value:
			raw = []byte(v.String())
		default:
			var err error
			raw, err = json.Marshal(v)
			if err != nil {
				return nil, err
			}
		}
		inputLen = int32(len(raw))
		inputAddr = i.evalHeapPtr
		heapPtr += inputLen
		copy(mem[inputAddr:inputAddr+inputLen], raw)

		metrics.Timer("wasm_vm_eval_prepare_input").Stop()
	}

	// Setting the ctx here ensures that it'll be available to builtins that
	// make use of it (e.g. `http.send`); and it will spawn a go routine
	// cancelling the builtins that use topdown.Cancel, when the context is
	// cancelled.
	i.dispatcher.Reset(ctx, seed, ns, iqbCache, ph)

	metrics.Timer("wasm_vm_eval_call").Start()
	resultAddr, err := i.evalOneOff(ctx, int32(entrypoint), i.dataAddr, inputAddr, inputLen, heapPtr)
	if err != nil {
		return nil, err
	}
	metrics.Timer("wasm_vm_eval_call").Stop()

	data := i.memory.UnsafeData(i.store)[resultAddr:]
	n := bytes.IndexByte(data, 0)
	if n < 0 {
		n = 0
	}

	// Skip free'ing input and result JSON as the heap will be reset next round anyway.
	return data[:n], nil
}

// evalCompat evaluates a policy using multiple calls into the VM to set the stage.
// It's been superceded with ABI version 1.2, but still here for compatibility with
// Wasm modules lacking the needed export (i.e., ABI 1.1).
func (i *VM) evalCompat(ctx context.Context,
	entrypoint int32,
	input *interface{},
	metrics metrics.Metrics,
	seed io.Reader,
	ns time.Time,
	iqbCache cache.InterQueryCache,
	ph print.Hook) ([]byte, error) {
	metrics.Timer("wasm_vm_eval").Start()
	defer metrics.Timer("wasm_vm_eval").Stop()

	metrics.Timer("wasm_vm_eval_prepare_input").Start()

	// Setting the ctx here ensures that it'll be available to builtins that
	// make use of it (e.g. `http.send`); and it will spawn a go routine
	// cancelling the builtins that use topdown.Cancel, when the context is
	// cancelled.
	i.dispatcher.Reset(ctx, seed, ns, iqbCache, ph)

	err := i.setHeapState(ctx, i.evalHeapPtr)
	if err != nil {
		return nil, err
	}

	// Parse the input JSON and activate it with the data.
	ctxAddr, err := i.evalCtxNew(ctx)
	if err != nil {
		return nil, err
	}

	if i.dataAddr != 0 {
		if err := i.evalCtxSetData(ctx, ctxAddr, i.dataAddr); err != nil {
			return nil, err
		}
	}

	if err := i.evalCtxSetEntrypoint(ctx, ctxAddr, int32(entrypoint)); err != nil {
		return nil, err
	}

	if input != nil {
		inputAddr, err := i.toRegoJSON(ctx, *input, false)
		if err != nil {
			return nil, err
		}

		if err := i.evalCtxSetInput(ctx, ctxAddr, inputAddr); err != nil {
			return nil, err
		}
	}
	metrics.Timer("wasm_vm_eval_prepare_input").Stop()

	// Evaluate the policy.
	metrics.Timer("wasm_vm_eval_execute").Start()
	err = i.eval(ctx, ctxAddr)
	metrics.Timer("wasm_vm_eval_execute").Stop()
	if err != nil {
		return nil, err
	}

	metrics.Timer("wasm_vm_eval_prepare_result").Start()
	resultAddr, err := i.evalCtxGetResult(ctx, ctxAddr)
	if err != nil {
		return nil, err
	}

	serialized, err := i.valueDump(ctx, resultAddr)
	if err != nil {
		return nil, err
	}

	data := i.memory.UnsafeData(i.store)[serialized:]
	n := bytes.IndexByte(data, 0)
	if n < 0 {
		n = 0
	}

	metrics.Timer("wasm_vm_eval_prepare_result").Stop()

	// Skip free'ing input and result JSON as the heap will be reset next round anyway.

	return data[0:n], nil
}

// SetPolicyData Will either update the VM's data or, if the policy changed,
// re-initialize the VM.
func (i *VM) SetPolicyData(ctx context.Context, opts vmOpts) error {

	if !bytes.Equal(opts.policy, i.policy) {
		// Swap the instance to a new one, with new policy.
		n, err := newVM(opts, i.engine)
		if err != nil {
			return err
		}

		*i = *n
		return nil
	}

	i.dataAddr = 0

	var err error
	if err = i.setHeapState(ctx, i.baseHeapPtr); err != nil {
		return err
	}

	if opts.parsedData != nil {
		if uint32(i.memory.DataSize(i.store))-uint32(i.baseHeapPtr) < uint32(len(opts.parsedData)) {
			delta := uint32(len(opts.parsedData)) - (uint32(i.memory.DataSize(i.store)) - uint32(i.baseHeapPtr))
			_, err := i.memory.Grow(i.store, uint64(util.Pages(delta)))
			if err != nil {
				return err
			}
		}
		mem := i.memory.UnsafeData(i.store)
		for src, dest := 0, i.baseHeapPtr; src < len(opts.parsedData); src, dest = src+1, dest+1 {
			mem[dest] = opts.parsedData[src]
		}
		i.dataAddr = opts.parsedDataAddr
		i.evalHeapPtr = i.baseHeapPtr + int32(len(opts.parsedData))
		err := i.setHeapState(ctx, i.evalHeapPtr)
		if err != nil {
			return err
		}
	} else if opts.data != nil {
		if i.dataAddr, err = i.toRegoJSON(ctx, opts.data, true); err != nil {
			return err
		}
	}

	if i.evalHeapPtr, err = i.getHeapState(ctx); err != nil {
		return err
	}

	return nil
}

type abortError struct {
	message string
}

type cancelledError struct {
	message string
}

// Println is invoked if the policy WASM code calls opa_println().
func (i *VM) Println(arg int32) {
	data := i.memory.UnsafeData(i.store)[arg:]
	n := bytes.IndexByte(data, 0)
	if n == -1 {
		panic("invalid opa_println argument")
	}

	fmt.Printf("opa_println(): %s\n", string(data[:n]))
}

type builtinError struct {
	err error
}

// Entrypoints returns a mapping of entrypoint name to ID for use by Eval().
func (i *VM) Entrypoints() map[string]int32 {
	return i.entrypointIDs
}

// SetDataPath will update the current data on the VM by setting the value at the
// specified path. If an error occurs the instance is still in a valid state, however
// the data will not have been modified.
func (i *VM) SetDataPath(ctx context.Context, path []string, value interface{}) error {
	// Reset the heap ptr before patching the vm to try and keep any
	// new allocations safe from subsequent heap resets on eval.
	err := i.setHeapState(ctx, i.evalHeapPtr)
	if err != nil {
		return err
	}

	valueAddr, err := i.toRegoJSON(ctx, value, true)
	if err != nil {
		return err
	}

	pathAddr, err := i.toRegoJSON(ctx, path, true)
	if err != nil {
		return err
	}

	result, err := i.valueAddPath(ctx, i.dataAddr, pathAddr, valueAddr)
	if err != nil {
		return err
	}

	// We don't need to free the value, assume it is "owned" as part of the
	// overall data object now.
	// We do need to free the path

	if err := i.free(ctx, pathAddr); err != nil {
		return err
	}

	// Update the eval heap pointer to accommodate for any new allocations done
	// while patching.
	i.evalHeapPtr, err = i.getHeapState(ctx)
	if err != nil {
		return err
	}

	errc := result
	if errc != 0 {
		return fmt.Errorf("unable to set data value for path %v, err=%d", path, errc)
	}

	return nil
}

// RemoveDataPath will update the current data on the VM by removing the value at the
// specified path. If an error occurs the instance is still in a valid state, however
// the data will not have been modified.
func (i *VM) RemoveDataPath(ctx context.Context, path []string) error {
	pathAddr, err := i.toRegoJSON(ctx, path, true)
	if err != nil {
		return err
	}

	errc, err := i.valueRemovePath(ctx, i.dataAddr, pathAddr)
	if err != nil {
		return err
	}

	if err := i.free(ctx, pathAddr); err != nil {
		return err
	}

	if errc != 0 {
		return fmt.Errorf("unable to set data value for path %v, err=%d", path, errc)
	}

	return nil
}

// fromRegoJSON parses serialized JSON from the Wasm memory buffer into
// native go types.
func (i *VM) fromRegoJSON(ctx context.Context, addr int32, free bool) (interface{}, error) {
	serialized, err := i.jsonDump(ctx, addr)
	if err != nil {
		return nil, err
	}

	data := i.memory.UnsafeData(i.store)[serialized:]
	n := bytes.IndexByte(data, 0)
	if n < 0 {
		n = 0
	}

	// Parse the result into go types.

	decoder := json.NewDecoder(bytes.NewReader(data[0:n]))
	decoder.UseNumber()

	var result interface{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	if free {
		if err := i.free(ctx, serialized); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// toRegoJSON converts go native JSON to Rego JSON. If the value is
// an AST type it will be dumped using its stringer.
func (i *VM) toRegoJSON(ctx context.Context, v interface{}, free bool) (int32, error) {
	var raw []byte
	switch v := v.(type) {
	case []byte:
		raw = v
	case *ast.Term:
		raw = []byte(v.String())
	case ast.Value:
		raw = []byte(v.String())
	default:
		var err error
		raw, err = json.Marshal(v)
		if err != nil {
			return 0, err
		}
	}

	n := int32(len(raw))
	p, err := i.malloc(ctx, n)
	if err != nil {
		return 0, err
	}

	copy(i.memory.UnsafeData(i.store)[p:p+n], raw)

	addr, err := i.valueParse(ctx, p, n)
	if err != nil {
		return 0, err
	}

	if free {
		if err := i.free(ctx, p); err != nil {
			return 0, err
		}
	}

	return addr, nil
}

func (i *VM) getHeapState(ctx context.Context) (int32, error) {
	return i.heapPtrGet(ctx)
}

func (i *VM) setHeapState(ctx context.Context, ptr int32) error {
	return i.heapPtrSet(ctx, ptr)
}

func (i *VM) cloneDataSegment() (int32, []byte) {
	// The parsed data values sit between the base heap address and end
	// at the eval heap pointer address.
	srcData := i.memory.UnsafeData(i.store)[i.baseHeapPtr:i.evalHeapPtr]
	patchedData := make([]byte, len(srcData))
	copy(patchedData, srcData)
	return i.dataAddr, patchedData
}

func call(ctx context.Context, vm *VM, name string, args ...int32) (int32, error) {
	res, err := callOrCancel(ctx, vm, name, args...)
	if err != nil {
		return 0, err
	}
	return res.(int32), nil
}

func callVoid(ctx context.Context, vm *VM, name string, args ...int32) error {
	_, err := callOrCancel(ctx, vm, name, args...)
	return err
}

func callOrCancel(ctx context.Context, vm *VM, name string, args ...int32) (interface{}, error) {
	sl := make([]interface{}, len(args))
	for i := range sl {
		sl[i] = args[i]
	}

	// `done` is closed when the eval is done;
	// `ctxdone` is used to ensure that this goroutine is not running rogue;
	// it may interact badly with other calls into this VM because of async
	// execution. Concretely, there's no guarantee which branch of done or
	// ctx.Done() is selected when they're both good to go. Hence, this may
	// interrupt the VM long after _this_ functions is done. By tying them
	// together (`<-ctxdone` at the end of callOrCancel, `close(ctxdone)`
	// here), we can avoid that.
	done := make(chan struct{})
	ctxdone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			vm.intHandle.Interrupt()
		case <-done:
		}
		close(ctxdone)
	}()

	f := vm.instance.GetFunc(vm.store, name)
	// If this call into the VM ends up calling host functions (builtins not
	// implemented in Wasm), and those panic, wasmtime will re-throw them,
	// and this is where we deal with that:
	res, err := func() (res interface{}, err error) {
		defer close(done)
		defer func() {
			if e := recover(); e != nil {
				switch e := e.(type) {
				case abortError:
					err = sdk_errors.New(sdk_errors.InternalErr, e.message)
				case cancelledError:
					err = sdk_errors.New(sdk_errors.CancelledErr, e.message)
				case builtinError:
					err = sdk_errors.New(sdk_errors.InternalErr, e.err.Error())
				default:
					panic(e)
				}
			}
		}()
		res, err = f.Call(vm.store, sl...)
		return
	}()
	if err != nil {
		// if last err was trap, extract information
		var t *wasmtime.Trap
		if errors.As(err, &t) {
			code := t.Code()
			if code != nil && *code == wasmtime.Interrupt {
				return 0, sdk_errors.New(sdk_errors.CancelledErr, getStack(t.Frames(), "interrupted"))
			}
			return 0, sdk_errors.New(sdk_errors.InternalErr, getStack(t.Frames(), "trapped"))
		}
		return 0, err
	}
	<-ctxdone // wait for the goroutine that's checking ctx
	return res, nil
}

func getStack(fs []*wasmtime.Frame, desc string) string {
	var b strings.Builder
	b.WriteString(desc)
	if len(fs) > 1 {
		b.WriteString(" at ")
		for i := len(fs) - 1; i >= 0; i-- { // backwards
			fr := fs[i]
			if fun := fr.FuncName(); fun != nil {
				if i != len(fs)-1 {
					b.WriteRune('/')
				}
				b.WriteString(*fun)

			}
		}
	}
	return b.String()
}
