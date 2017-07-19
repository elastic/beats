package monitoring

import (
	"encoding/json"
	"expvar"
	"math"
	"strconv"
	"sync"

	"github.com/elastic/beats/libbeat/common/atomic"
)

// makeExpvar wraps a callback for registering a metrics with expvar.Publish.
type makeExpvar func() string

// Int is a 64 bit integer variable satisfying the Var interface.
type Int struct{ i atomic.Int64 }

// NewInt creates and registers a new integer variable.
//
// Note: If the registry is configured to publish variables to expvar, the
// variable will be available via expvars package as well, but can not be removed
// anymore.
func NewInt(r *Registry, name string, opts ...Option) *Int {
	if r == nil {
		r = Default
	}

	v := &Int{}
	addVar(r, name, opts, v, makeExpvar(func() string {
		return strconv.FormatInt(v.Get(), 10)
	}))
	return v
}

func (v *Int) Get() int64               { return v.i.Load() }
func (v *Int) Set(value int64)          { v.i.Store(value) }
func (v *Int) Add(delta int64)          { v.i.Add(delta) }
func (v *Int) Sub(delta int64)          { v.i.Sub(delta) }
func (v *Int) Inc()                     { v.i.Inc() }
func (v *Int) Dec()                     { v.i.Dec() }
func (v *Int) Visit(_ Mode, vs Visitor) { vs.OnInt(v.Get()) }

// Uint is a 64bit unsigned integer variable satisfying the Var interface.
type Uint struct{ u atomic.Uint64 }

// NewUint creates and registers a new unsigned integer variable.
//
// Note: If the registry is configured to publish variables to expvar, the
// variable will be available via expvars package as well, but can not be removed
// anymore.
func NewUint(r *Registry, name string, opts ...Option) *Uint {
	if r == nil {
		r = Default
	}

	v := &Uint{}
	addVar(r, name, opts, v, makeExpvar(func() string {
		return strconv.FormatUint(v.Get(), 10)
	}))
	return v
}

func (v *Uint) Get() uint64      { return v.u.Load() }
func (v *Uint) Set(value uint64) { v.u.Store(value) }
func (v *Uint) Add(delta uint64) { v.u.Add(delta) }
func (v *Uint) Sub(delta uint64) { v.u.Sub(delta) }
func (v *Uint) Inc()             { v.u.Inc() }
func (v *Uint) Dec()             { v.u.Dec() }
func (v *Uint) Visit(_ Mode, vs Visitor) {
	value := v.Get() & (^uint64(1 << 63))
	vs.OnInt(int64(value))
}

// Float is a 64 bit float variable satisfying the Var interface.
type Float struct{ f atomic.Uint64 }

// NewFloat creates and registers a new float variable.
//
// Note: If the registry is configured to publish variables to expvar, the
// variable will be available via expvars package as well, but can not be removed
// anymore.
func NewFloat(r *Registry, name string, opts ...Option) *Float {
	if r == nil {
		r = Default
	}

	v := &Float{}
	addVar(r, name, opts, v, makeExpvar(func() string {
		return strconv.FormatFloat(v.Get(), 'g', -1, 64)
	}))
	return v
}

func (v *Float) Get() float64             { return math.Float64frombits(v.f.Load()) }
func (v *Float) Set(value float64)        { v.f.Store(math.Float64bits(value)) }
func (v *Float) Sub(delta float64)        { v.Add(-delta) }
func (v *Float) Visit(_ Mode, vs Visitor) { vs.OnFloat(v.Get()) }

func (v *Float) Add(delta float64) {
	for {
		cur := v.f.Load()
		next := math.Float64bits(math.Float64frombits(cur) + delta)
		if v.f.CAS(cur, next) {
			return
		}
	}
}

// String is a string variable satisfying the Var interface.
type String struct {
	mu sync.RWMutex
	s  string
}

// NewString creates and registers a new string variable.
//
// Note: If the registry is configured to publish variables to expvar, the
// variable will be available via expvars package as well, but can not be removed
// anymore.
func NewString(r *Registry, name string, opts ...Option) *String {
	if r == nil {
		r = Default
	}

	v := &String{}
	addVar(r, name, opts, v, makeExpvar(func() string {
		b, _ := json.Marshal(v.Get())
		return string(b)
	}))
	return v
}

func (v *String) Visit(_ Mode, vs Visitor) {
	vs.OnString(v.Get())
}

func (v *String) Get() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.s
}

func (v *String) Set(s string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.s = s
}

func (v *String) Clear() {
	v.Set("")
}

func (v *String) Fail(err error) {
	v.Set(err.Error())
}

type FuncVar func(Mode, Visitor)

func (f FuncVar) Visit(m Mode, vs Visitor) { f(m, vs) }

type Func struct {
	f FuncVar
}

func NewFunc(r *Registry, name string, f func(Mode, Visitor), opts ...Option) *Func {
	if r == nil {
		r = Default
	}

	v := &Func{f}
	addVar(r, name, opts, v, nil)
	return v
}

func (f *Func) Visit(m Mode, vs Visitor) { f.f(m, vs) }

func (m makeExpvar) String() string { return m() }

func addVar(r *Registry, name string, opts []Option, v Var, ev expvar.Var) {
	O := varOpts(r.opts, opts)
	r.doAdd(name, v, O)
	if O.publishExpvar && ev != nil {
		expvar.Publish(fullName(r, name), ev)
	}
}

func fullName(r *Registry, name string) string {
	if r.name == "" {
		return name
	}
	return r.name + "." + name
}
