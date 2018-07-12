package skima

import (
	"strings"

	"testing"

	"github.com/stretchr/testify/assert"

	"fmt"

	"github.com/elastic/beats/libbeat/common"
)

type ValueResult struct {
	valid   bool
	message string // Reason this is invalid
}

var ValidValueResult = ValueResult{true, ""}

type Checker func(v interface{}) ValueResult

type IsDef struct {
	name    string
	checker Checker
}

func Is(name string, checker Checker) IsDef {
	return IsDef{name, checker}
}

type ValueValidator = func(isdef IsDef) ValueResult

type Map map[string]interface{}

type StrictMap Map

type Validator func(common.MapStr) MapResults

type MapResults map[string][]ValueResult

func (r MapResults) RecordResult(path string, result ValueResult) {
	if r[path] == nil {
		r[path] = []ValueResult{result}
	} else {
		r[path] = append(r[path], result)
	}
}

func (r MapResults) eachResult(f func(string, ValueResult)) {
	for path, pathResults := range r {
		for _, result := range pathResults {
			f(path, result)
		}
	}
}

func (r MapResults) Errors() MapResults {
	errors := MapResults{}
	r.eachResult(func(path string, vr ValueResult) {
		if !vr.valid {
			errors.RecordResult(path, vr)
		}
	})
	return errors
}

func (r MapResults) Valid() bool {
	return len(r.Errors()) == 0
}

/*
func combineResults(results []*MapResults) *MapResults {
	// Use sets to de-dupe these
	output := MapResult{testedFields: map[string]struct{}{}}

	for _, res := range results {
		for k, _ := range res.testedFields {
			output.testedFields[k] = struct{}{}
		}
	}

	return &output
}

func Compose(validators ...Validator) Validator {
	return func(actual common.MapStr) *MapResult {
		results := make([]*MapResult, len(validators))
		for idx, validator := range validators {
			results[idx] = validator(actual)
		}
		return combineResults(results)
	}
}

func Strict(validator Validator) Validator {
	return func(actual common.MapStr) *MapResult {
		res := validator(actual)

		missed := map[string]struct{}{}

		walk(actual, func(
			key string,
			value interface{},
			currentMap common.MapStr,
			rootMap common.MapStr,
			path []string,
			dottedPath string) {
			if _, ok := res.testedFields[dottedPath]; !ok {
				missed[dottedPath] = struct{}{}
			}
		})

		//assert.Empty(t, missed, "Unexpected fields found during strict schema test")

		return res
	}
}
*/

func Test(t *testing.T, r MapResults) {
	assert.True(t, r.Valid())
	r.Errors().eachResult(func(p string, vr ValueResult) {
		msg := fmt.Sprintf("%s: %s", p, vr.message)
		assert.True(t, vr.valid, msg)
	})
}

func Schema(expected Map) Validator {
	return func(actual common.MapStr) MapResults {
		return Validate(expected, actual)
	}
}

func Validate(expected Map, actual common.MapStr) MapResults {
	return walkValidate(expected, actual)
}

type WalkObserver func(
	key string,
	value interface{},
	currentMap common.MapStr,
	rootMap common.MapStr,
	path []string,
	dottedPath string,
)

func walk(m common.MapStr, wo WalkObserver) {
	walkFull(m, m, []string{}, wo)
}

// TODO: Handle slices/arrays. We intentionally don't handle list types now because we don't need it (yet)
// and it isn't clear in the context of validation what the right thing is to do there beyond letting the user
// perform a custom validation
func walkFull(m common.MapStr, root common.MapStr, path []string, wo WalkObserver) {
	for k, v := range m {
		splitK := strings.Split(k, ".")
		var newPath []string
		newPath = append(newPath, path...)
		newPath = append(newPath, splitK...)

		dottedPath := strings.Join(newPath, ".")

		wo(k, v, m, root, newPath, dottedPath)

		// Walk nested maps
		vIsMap := false
		var mapV common.MapStr
		// Note that we intentionally do not handle StrictMap
		// In this branching conditional! That is handled by
		// Initializing a whole new validation chain in a lower spot
		if convertedMS, ok := v.(common.MapStr); ok {
			mapV = convertedMS
			vIsMap = true
		} else if convertedM, ok := v.(Map); ok {
			mapV = common.MapStr(convertedM)
			vIsMap = true
		}

		if vIsMap {
			walkFull(mapV, root, newPath, wo)
		}
	}
}

func walkValidate(expected Map, actual common.MapStr) (results MapResults) {
	results = MapResults{}
	walk(
		common.MapStr(expected),
		func(expectedK string,
			expectedV interface{},
			currentMap common.MapStr,
			rootMap common.MapStr,
			path []string,
			dottedPath string) {

			actualV, _ := actual.GetValue(dottedPath)

			/*else if sm, isStrictMap := expectedV.(StrictMap); isStrictMap {
					if actualM, ok := actualV.(common.MapStr); ok {
						Strict(Schema(Map(sm)))(actualM)
					} else {
						//assert.Fail(t, "Expected %s to be a strictly defined map, but it was actually '%v'", dottedPath, actualV)
					}
			} */

			isDef, isIsDef := expectedV.(IsDef)
			if isIsDef {
				results.RecordResult(dottedPath, isDef.checker(actualV))
			} else if _, isMap := expectedV.(Map); !isMap {
				results.RecordResult(dottedPath, IsEqual(expectedV).checker(actualV))
			}
		})

	return results
}

func isMapType(v interface{}) bool {
	_, isMap := v.(Map)
	if isMap {
		return isMap
	}
	_, isStrictMap := v.(StrictMap)
	return isStrictMap
}
