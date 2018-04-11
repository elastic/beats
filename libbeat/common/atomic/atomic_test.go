package atomic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtomicBool(t *testing.T) {
	assert := assert.New(t)

	var b Bool
	assert.False(b.Load(), "check zero value is false")

	b = MakeBool(true)
	assert.True(b.Load(), "check value initializer with 'true' value")

	b.Store(false)
	assert.False(b.Load(), "check store to false")

	old := b.Swap(true)
	assert.False(old, "check old value of swap operation is 'false'")
	assert.True(b.Load(), "check new value after swap is 'true'")

	old = b.Swap(false)
	assert.True(old, "check old value of second swap operation is 'true'")
	assert.False(b.Load(), "check new value after second swap is 'false'")

	ok := b.CAS(true, true)
	assert.False(ok, "check CAS fails with wrong 'old' value")
	assert.False(b.Load(), "check failed CAS did not change value 'false'")

	ok = b.CAS(false, true)
	assert.True(ok, "check CAS succeeds with correct 'old' value")
	assert.True(b.Load(), "check CAS did change value to 'true'")
}

func TestAtomicInt32(t *testing.T) {
	assert := assert.New(t)
	check := func(expected, actual int32, msg string) {
		assert.Equal(expected, actual, msg)
	}

	var v Int32
	check(0, v.Load(), "check zero value")

	v = MakeInt32(23)
	check(23, v.Load(), "check value initializer")

	v.Store(42)
	check(42, v.Load(), "check store new value")

	new := v.Inc()
	check(43, new, "check increment returns new value")
	check(43, v.Load(), "check increment did store new value")

	new = v.Dec()
	check(42, new, "check decrement returns new value")
	check(42, v.Load(), "check decrement did store new value")

	new = v.Add(8)
	check(50, new, "check add returns new value")
	check(50, v.Load(), "check add did store new value")

	new = v.Sub(8)
	check(42, new, "check sub returns new value")
	check(42, v.Load(), "check sub did store new value")

	old := v.Swap(101)
	check(42, old, "check swap returns old value")
	check(101, v.Load(), "check swap stores new value")

	ok := v.CAS(0, 23)
	assert.False(ok, "check CAS with wrong old value fails")
	check(101, v.Load(), "check failed CAS did not change value")

	ok = v.CAS(101, 23)
	assert.True(ok, "check CAS succeeds")
	check(23, v.Load(), "check CAS did store new value")
}

func TestAtomicInt64(t *testing.T) {
	assert := assert.New(t)
	check := func(expected, actual int64, msg string) {
		assert.Equal(expected, actual, msg)
	}

	var v Int64
	check(0, v.Load(), "check zero value")

	v = MakeInt64(23)
	check(23, v.Load(), "check value initializer")

	v.Store(42)
	check(42, v.Load(), "check store new value")

	new := v.Inc()
	check(43, new, "check increment returns new value")
	check(43, v.Load(), "check increment did store new value")

	new = v.Dec()
	check(42, new, "check decrement returns new value")
	check(42, v.Load(), "check decrement did store new value")

	new = v.Add(8)
	check(50, new, "check add returns new value")
	check(50, v.Load(), "check add did store new value")

	new = v.Sub(8)
	check(42, new, "check sub returns new value")
	check(42, v.Load(), "check sub did store new value")

	old := v.Swap(101)
	check(42, old, "check swap returns old value")
	check(101, v.Load(), "check swap stores new value")

	ok := v.CAS(0, 23)
	assert.False(ok, "check CAS with wrong old value fails")
	check(101, v.Load(), "check failed CAS did not change value")

	ok = v.CAS(101, 23)
	assert.True(ok, "check CAS succeeds")
	check(23, v.Load(), "check CAS did store new value")
}

func TestAtomicUint32(t *testing.T) {
	assert := assert.New(t)
	check := func(expected, actual uint32, msg string) {
		assert.Equal(expected, actual, msg)
	}

	var v Uint32
	check(0, v.Load(), "check zero value")

	v = MakeUint32(23)
	check(23, v.Load(), "check value initializer")

	v.Store(42)
	check(42, v.Load(), "check store new value")

	new := v.Inc()
	check(43, new, "check increment returns new value")
	check(43, v.Load(), "check increment did store new value")

	new = v.Dec()
	check(42, new, "check decrement returns new value")
	check(42, v.Load(), "check decrement did store new value")

	new = v.Add(8)
	check(50, new, "check add returns new value")
	check(50, v.Load(), "check add did store new value")

	new = v.Sub(8)
	check(42, new, "check sub returns new value")
	check(42, v.Load(), "check sub did store new value")

	old := v.Swap(101)
	check(42, old, "check swap returns old value")
	check(101, v.Load(), "check swap stores new value")

	ok := v.CAS(0, 23)
	assert.False(ok, "check CAS with wrong old value fails")
	check(101, v.Load(), "check failed CAS did not change value")

	ok = v.CAS(101, 23)
	assert.True(ok, "check CAS succeeds")
	check(23, v.Load(), "check CAS did store new value")
}

func TestAtomicUint64(t *testing.T) {
	assert := assert.New(t)
	check := func(expected, actual uint64, msg string) {
		assert.Equal(expected, actual, msg)
	}

	var v Uint64
	check(0, v.Load(), "check zero value")

	v = MakeUint64(23)
	check(23, v.Load(), "check value initializer")

	v.Store(42)
	check(42, v.Load(), "check store new value")

	new := v.Inc()
	check(43, new, "check increment returns new value")
	check(43, v.Load(), "check increment did store new value")

	new = v.Dec()
	check(42, new, "check decrement returns new value")
	check(42, v.Load(), "check decrement did store new value")

	new = v.Add(8)
	check(50, new, "check add returns new value")
	check(50, v.Load(), "check add did store new value")

	new = v.Sub(8)
	check(42, new, "check sub returns new value")
	check(42, v.Load(), "check sub did store new value")

	old := v.Swap(101)
	check(42, old, "check swap returns old value")
	check(101, v.Load(), "check swap stores new value")

	ok := v.CAS(0, 23)
	assert.False(ok, "check CAS with wrong old value fails")
	check(101, v.Load(), "check failed CAS did not change value")

	ok = v.CAS(101, 23)
	assert.True(ok, "check CAS succeeds")
	check(23, v.Load(), "check CAS did store new value")
}

func TestAtomicUint(t *testing.T) {
	assert := assert.New(t)
	check := func(expected, actual uint, msg string) {
		assert.Equal(expected, actual, msg)
	}

	var v Uint
	check(0, v.Load(), "check zero value")

	v = MakeUint(23)
	check(23, v.Load(), "check value initializer")

	v.Store(42)
	check(42, v.Load(), "check store new value")

	new := v.Inc()
	check(43, new, "check increment returns new value")
	check(43, v.Load(), "check increment did store new value")

	new = v.Dec()
	check(42, new, "check decrement returns new value")
	check(42, v.Load(), "check decrement did store new value")

	new = v.Add(8)
	check(50, new, "check add returns new value")
	check(50, v.Load(), "check add did store new value")

	new = v.Sub(8)
	check(42, new, "check sub returns new value")
	check(42, v.Load(), "check sub did store new value")

	old := v.Swap(101)
	check(42, old, "check swap returns old value")
	check(101, v.Load(), "check swap stores new value")

	ok := v.CAS(0, 23)
	assert.False(ok, "check CAS with wrong old value fails")
	check(101, v.Load(), "check failed CAS did not change value")

	ok = v.CAS(101, 23)
	assert.True(ok, "check CAS succeeds")
	check(23, v.Load(), "check CAS did store new value")
}
