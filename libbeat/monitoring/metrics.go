package monitoring

import (
	"math"
	"sync"
	"sync/atomic"
)

// makeExpvar wraps a callback for registering a metrics with expvar.Publish.
type makeExpvar func() string

// Int is a 64 bit integer variable satisfying the Var interface.
type Int struct{ i int64 }

// NewInt registers a new global integer metrics.
func NewInt(name string) *Int {
	return Default.NewInt(name)
}

func (v *Int) Visit(vs Visitor) error { return vs.OnInt(v.Get()) }
func (v *Int) Get() int64             { return atomic.LoadInt64(&v.i) }
func (v *Int) Set(value int64)        { atomic.StoreInt64(&v.i, value) }
func (v *Int) Add(delta int64)        { atomic.AddInt64(&v.i, delta) }
func (v *Int) Inc()                   { atomic.AddInt64(&v.i, 1) }
func (v *Int) Dec()                   { atomic.AddInt64(&v.i, -1) }

// Float is a 64 bit float variable satisfying the Var interface.
type Float struct{ f uint64 }

// NewFloat registers a new global floating point metric.
func NewFloat(name string) *Float {
	return Default.NewFloat(name)
}

func (v *Float) Visit(vs Visitor) error { return vs.OnFloat(v.Get()) }
func (v *Float) Get() float64           { return math.Float64frombits(atomic.LoadUint64(&v.f)) }
func (v *Float) Set(value float64)      { atomic.StoreUint64(&v.f, math.Float64bits(value)) }
func (v *Float) Sub(delta float64)      { v.Add(-delta) }

func (v *Float) Add(delta float64) {
	for {
		cur := atomic.LoadUint64(&v.f)
		next := math.Float64bits(math.Float64frombits(cur) + delta)
		if atomic.CompareAndSwapUint64(&v.f, cur, next) {
			return
		}
	}
}

// String is a string variable satisfying the Var interface.
type String struct {
	mu sync.RWMutex
	s  string
}

// NewString registers a new global string metric.
func NewString(name string) *String {
	return Default.NewString(name)
}

func (v *String) Visit(vs Visitor) error { return vs.OnString(v.Get()) }

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

func (m makeExpvar) String() string { return m() }
