package isdef

import (
	"fmt"
	"reflect"

	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
)

// IsEqual tests that the given object is equal to the actual object.
func IsEqual(to interface{}) IsDef {
	toV := reflect.ValueOf(to)
	isDefFactory, ok := equalChecks[toV.Type()]

	// If there are no handlers declared explicitly for this type we perform a deep equality check
	if !ok {
		return IsDeepEqual(to)
	}

	// We know this is an isdef due to the Register check previously
	checker := isDefFactory.Call([]reflect.Value{toV})[0].Interface().(IsDef).Checker

	return Is("equals", func(path llpath.Path, v interface{}) *llresult.Results {
		return checker(path, v)
	})
}

// KeyPresent checks that the given key is in the map, even if it has a nil value.
var KeyPresent = IsDef{Name: "check key present"}

// KeyMissing checks that the given key is not present defined.
var KeyMissing = IsDef{Name: "check key not present", CheckKeyMissing: true}

func init() {
	MustRegisterEqual(IsEqualToTime)
}

// InvalidEqualFnError is the error type returned by RegisterEqual when
// there is an issue with the given function.
type InvalidEqualFnError struct{ msg string }

func (e InvalidEqualFnError) Error() string {
	return fmt.Sprintf("Function is not a valid equal function: %s", e.msg)
}

// MustRegisterEqual is the panic-ing equivalent of RegisterEqual.
func MustRegisterEqual(fn interface{}) {
	if err := RegisterEqual(fn); err != nil {
		panic(fmt.Sprintf("Could not register fn as equal! %v", err))
	}
}

var equalChecks = map[reflect.Type]reflect.Value{}

// RegisterEqual takes a function of the form fn(v someType) IsDef
// and registers it to check equality for that type.
func RegisterEqual(fn interface{}) error {
	fnV := reflect.ValueOf(fn)
	fnT := fnV.Type()

	if fnT.Kind() != reflect.Func {
		return InvalidEqualFnError{"Provided value is not a function"}
	}
	if fnT.NumIn() != 1 {
		return InvalidEqualFnError{"Equal FN should take one argument"}
	}
	if fnT.NumOut() != 1 {
		return InvalidEqualFnError{"Equal FN should return one value"}
	}
	if fnT.Out(0) != reflect.TypeOf(IsDef{}) {
		return InvalidEqualFnError{"Equal FN should return an IsDef"}
	}

	inT := fnT.In(0)
	if _, ok := equalChecks[inT]; ok {
		return InvalidEqualFnError{fmt.Sprintf("Duplicate Equal FN for type %v encountered!", inT)}
	}

	equalChecks[inT] = fnV

	return nil
}

// IsDeepEqual checks equality using reflect.DeepEqual.
func IsDeepEqual(to interface{}) IsDef {
	return Is("equals", func(path llpath.Path, v interface{}) *llresult.Results {
		if reflect.DeepEqual(v, to) {
			return llresult.ValidResult(path)
		}
		return llresult.SimpleResult(
			path,
			false,
			fmt.Sprintf("objects not equal: actual(%T(%v)) != expected(%T(%v))", v, v, to, to),
		)
	})
}

// IsNil tests that a value is nil.
var IsNil = Is("is nil", func(path llpath.Path, v interface{}) *llresult.Results {
	if v == nil {
		return llresult.ValidResult(path)
	}
	return llresult.SimpleResult(
		path,
		false,
		fmt.Sprintf("Value %#v is not nil", v),
	)
})
