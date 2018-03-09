package structform

// Visitor interface for iterating some structured input
type Visitor interface {
	ObjectVisitor
	ArrayVisitor
	ValueVisitor
}

type ExtVisitor interface {
	Visitor
	ArrayValueVisitor
	ObjectValueVisitor
	StringRefVisitor
}

//go:generate stringer -type=BaseType
type BaseType uint8

const (
	AnyType BaseType = iota
	ByteType
	StringType
	BoolType
	ZeroType
	IntType
	Int8Type
	Int16Type
	Int32Type
	Int64Type
	UintType
	Uint8Type
	Uint16Type
	Uint32Type
	Uint64Type
	Float32Type
	Float64Type
)

// ObjectVisitor iterates all fields in a dictionary like structure
type ObjectVisitor interface {
	OnObjectStart(len int, baseType BaseType) error
	OnObjectFinished() error
	OnKey(s string) error
}

// ArrayVisitor iterates all entries in a list/array like structure
type ArrayVisitor interface {
	OnArrayStart(len int, baseType BaseType) error
	OnArrayFinished() error
}

// ValueVisitor reports actual values in a structure being iterated
type ValueVisitor interface {
	OnNil() error

	OnBool(b bool) error

	OnString(s string) error

	// int
	OnInt8(i int8) error
	OnInt16(i int16) error
	OnInt32(i int32) error
	OnInt64(i int64) error
	OnInt(i int) error

	// uint
	OnByte(b byte) error
	OnUint8(u uint8) error
	OnUint16(u uint16) error
	OnUint32(u uint32) error
	OnUint64(u uint64) error
	OnUint(u uint) error

	// float
	OnFloat32(f float32) error
	OnFloat64(f float64) error
}

// ArrayValueVisitor passes arrays with known type. Implementation
// of ArrayValueVisitor is optional.
type ArrayValueVisitor interface {
	OnBoolArray([]bool) error

	OnStringArray([]string) error

	// int
	OnInt8Array([]int8) error
	OnInt16Array([]int16) error
	OnInt32Array([]int32) error
	OnInt64Array([]int64) error
	OnIntArray([]int) error

	// uint
	OnBytes([]byte) error
	OnUint8Array([]uint8) error
	OnUint16Array([]uint16) error
	OnUint32Array([]uint32) error
	OnUint64Array([]uint64) error
	OnUintArray([]uint) error

	// float
	OnFloat32Array([]float32) error
	OnFloat64Array([]float64) error
}

// ObjectValueVisitor passes map[string]T. Implementation
// of ObjectValueVisitor is optional.
type ObjectValueVisitor interface {
	OnBoolObject(map[string]bool) error

	OnStringObject(map[string]string) error

	// int
	OnInt8Object(map[string]int8) error
	OnInt16Object(map[string]int16) error
	OnInt32Object(map[string]int32) error
	OnInt64Object(map[string]int64) error
	OnIntObject(map[string]int) error

	// uint
	OnUint8Object(map[string]uint8) error
	OnUint16Object(map[string]uint16) error
	OnUint32Object(map[string]uint32) error
	OnUint64Object(map[string]uint64) error
	OnUintObject(map[string]uint) error

	// float
	OnFloat32Object(map[string]float32) error
	OnFloat64Object(map[string]float64) error
}

// StringRefVisitor handles strings by reference into a byte string.
// The reference must be processed immediately, as the string passed
// might get overwritten after the callback returns.
type StringRefVisitor interface {
	OnStringRef(s []byte) error
	OnKeyRef(s []byte) error
}

type extVisitor struct {
	Visitor
	ObjectValueVisitor
	ArrayValueVisitor
	StringRefVisitor
}

func EnsureExtVisitor(v Visitor) ExtVisitor {
	if ev, ok := v.(ExtVisitor); ok {
		return ev
	}

	e := &extVisitor{
		Visitor: v,
	}
	if ov, ok := v.(ObjectValueVisitor); ok {
		e.ObjectValueVisitor = ov
	} else {
		e.ObjectValueVisitor = extObjVisitor{v}
	}
	if av, ok := v.(ArrayValueVisitor); ok {
		e.ArrayValueVisitor = av
	} else {
		e.ArrayValueVisitor = extArrVisitor{v}
	}
	e.StringRefVisitor = MakeStringRefVisitor(v)

	return e
}
