package isdef

import (
	"fmt"
	"reflect"

	"github.com/elastic/go-lookslike/internal/llreflect"
	"github.com/elastic/go-lookslike/llpath"
	"github.com/elastic/go-lookslike/llresult"
	"github.com/elastic/go-lookslike/validator"
)

// Is creates a named IsDef with the given Checker.
func Is(name string, checker ValueValidator) IsDef {
	return IsDef{Name: name, Checker: checker}
}

// A ValueValidator is used to validate a value in an interface{}.
type ValueValidator func(path llpath.Path, v interface{}) *llresult.Results

// An IsDef defines the type of Check to do.
// Generally only Name and Checker are set. Optional and CheckKeyMissing are
// needed for weird checks like key presence.
type IsDef struct {
	Name            string
	Checker         ValueValidator
	Optional        bool
	CheckKeyMissing bool
}

// Check runs the IsDef at the given value at the given path
func (id IsDef) Check(path llpath.Path, v interface{}, keyExists bool) *llresult.Results {
	if id.CheckKeyMissing {
		if !keyExists {
			return llresult.ValidResult(path)
		}

		return llresult.SimpleResult(path, false, "this key should not exist")
	}

	if !id.Optional && !keyExists {
		return llresult.KeyMissingResult(path)
	}

	if id.Checker != nil {
		return id.Checker(path, v)
	}

	return llresult.ValidResult(path)
}

// Optional wraps an IsDef to mark the field's presence as Optional.
func Optional(id IsDef) IsDef {
	id.Name = "Optional " + id.Name
	id.Optional = true
	return id
}

// IsSliceOf validates that the array at the given key is an array of objects all validatable
// via the given validator.Validator.
func IsSliceOf(validator validator.Validator) IsDef {
	return Is("slice", func(path llpath.Path, v interface{}) *llresult.Results {
		if reflect.TypeOf(v).Kind() != reflect.Slice {
			return llresult.SimpleResult(path, false, "Expected slice at given path")
		}
		vSlice := llreflect.InterfaceToSliceOfInterfaces(v)

		res := llresult.NewResults()

		for idx, curV := range vSlice {
			var validatorRes *llresult.Results
			validatorRes = validator(curV)
			res.MergeUnderPrefix(path.ExtendSlice(idx), validatorRes)
		}

		return res
	})
}

// IsAny takes a variable number of IsDef's and combines them with a logical OR. If any single definition
// matches the key will be marked as valid.
func IsAny(of ...IsDef) IsDef {
	names := make([]string, len(of))
	for i, def := range of {
		names[i] = def.Name
	}
	isName := fmt.Sprintf("either %#v", names)

	return Is(isName, func(path llpath.Path, v interface{}) *llresult.Results {
		for _, def := range of {
			vr := def.Check(path, v, true)
			if vr.Valid {
				return vr
			}
		}

		return llresult.SimpleResult(
			path,
			false,
			fmt.Sprintf("Value was none of %#v, actual value was %#v", names, v),
		)
	})
}

// IsUnique instances are used in multiple spots, flagging a value as being in error if it's seen across invocations.
// To use it, assign IsUnique to a variable, then use that variable multiple times in a map[string]interface{}.
func IsUnique() IsDef {
	return ScopedIsUnique().IsUniqueTo("")
}

// UniqScopeTracker is represents the tracking data for invoking IsUniqueTo.
type UniqScopeTracker map[interface{}]string

// IsUniqueTo validates that the given value is only ever seen within a single namespace.
func (ust UniqScopeTracker) IsUniqueTo(namespace string) IsDef {
	return Is("unique", func(path llpath.Path, v interface{}) *llresult.Results {
		for trackerK, trackerNs := range ust {
			hasNamespace := len(namespace) > 0
			if reflect.DeepEqual(trackerK, v) && (!hasNamespace || namespace != trackerNs) {
				return llresult.SimpleResult(path, false, "Value '%v' is repeated", v)
			}
		}

		ust[v] = namespace
		return llresult.ValidResult(path)
	})
}

// ScopedIsUnique returns a new scope for uniqueness checks.
func ScopedIsUnique() UniqScopeTracker {
	return UniqScopeTracker{}
}
