package metrics

// OptUint is a wrapper for "optional" types, with the bool value indicating
// if the stored int is a legitimate value.
type OptUint struct {
	exists bool
	value  uint64
}

// NewUint returns a new uint wrapper
func NewUint() OptUint {
	return OptUint{
		exists: false,
		value:  0,
	}
}

// None marks the Uint as not having a value.
func (opt *OptUint) None() {
	opt.exists = false
}

// Exists returns true if the underlying value is valid
func (opt OptUint) Exists() bool {
	return opt.exists
}

// Some Sets a valid value inside the OptUint
func (opt *OptUint) Some(i uint64) {
	opt.value = i
	opt.exists = true
}

// ValueOrZero returns the stored value, or zero
// Please do not use this for populating reported data,
// as we actually want to avoid sending zeros where values are functionally null
func (opt OptUint) ValueOrZero() uint64 {
	if opt.exists {
		return opt.value
	}
	return 0
}

// OptFloat is a wrapper for "optional" types, with the bool value indicating
// if the stored int is a legitimate value.
type OptFloat struct {
	exists bool
	value  float64
}

// NewFloat returns a new uint wrapper
func NewFloat() OptFloat {
	return OptFloat{
		exists: false,
		value:  0,
	}
}

// None marks the Uint as not having a value.
func (opt *OptFloat) None() {
	opt.exists = false
}

// Some Sets a valid value inside the OptUint
func (opt *OptFloat) Some(i float64) {
	opt.value = i
	opt.exists = true
}

// Exists returns true if the underlying value is valid
func (opt OptFloat) Exists() bool {
	return opt.exists
}

// ValueOrZero returns the stored value, or zero
// Please do not use this for populating reported data,
// as we actually want to avoid sending zeros where values are functionally null
func (opt OptFloat) ValueOrZero() float64 {
	if opt.exists {
		return opt.value
	}
	return 0
}
