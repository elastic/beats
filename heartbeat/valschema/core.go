package valschema

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

type ValueValidator = func(t *testing.T, keyExists bool, actual interface{})

type Map map[string]interface{}

type StrictMap Map

type Validator func(*testing.T, common.MapStr) *ValidationResult

type ValidationResult struct {
	valid        bool
	testedFields map[string]struct{}
}

func combineResults(results []*ValidationResult) *ValidationResult {
	// Use sets to de-dupe these
	output := ValidationResult{testedFields: map[string]struct{}{}}

	for _, res := range results {
		for k, _ := range res.testedFields {
			output.testedFields[k] = struct{}{}
		}
	}

	return &output
}

func Compose(validators ...Validator) Validator {
	return func(t *testing.T, actual common.MapStr) *ValidationResult {
		results := make([]*ValidationResult, len(validators))
		for idx, validator := range validators {
			results[idx] = validator(t, actual)
		}
		return combineResults(results)
	}
}

func Strict(validator Validator) Validator {
	return func(t *testing.T, actual common.MapStr) *ValidationResult {
		res := validator(t, actual)

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

		assert.Empty(t, missed, "Unexpected fields found during strict schema test")

		return res
	}
}

func Schema(expected Map) Validator {
	return func(t *testing.T, actual common.MapStr) *ValidationResult {
		return Validate(t, expected, actual)
	}
}

func Validate(t *testing.T, expected Map, actual common.MapStr) *ValidationResult {
	return walkValidate(t, expected, actual)
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
		newPath := make([]string, len(path)+1)
		copy(newPath, path)
		newPath[len(path)] = k // Append the key

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

func walkValidate(t *testing.T, expected Map, actual common.MapStr) (output *ValidationResult) {
	output = &ValidationResult{testedFields: map[string]struct{}{}}
	walk(
		common.MapStr(expected),
		func(expectedK string,
			expectedV interface{},
			currentMap common.MapStr,
			rootMap common.MapStr,
			path []string,
			dottedPath string) {
			actualHasKey, _ := actual.HasKey(dottedPath)
			if actualHasKey {
				output.testedFields[dottedPath] = struct{}{}
			}

			actualV, _ := actual.GetValue(dottedPath)

			vv, isVV := expectedV.(ValueValidator)
			if isVV {
				vv(t, actualHasKey, actualV)
			} else if sm, isStrictMap := expectedV.(StrictMap); isStrictMap {
				if actualM, ok := actualV.(common.MapStr); ok {
					Strict(Schema(Map(sm)))(t, actualM)
				} else {
					assert.Fail(t, "Expected %s to be a strictly defined map, but it was actually '%v'", dottedPath, actualV)
				}
			} else if _, isMap := expectedV.(Map); !isMap {
				assert.Equal(t, expectedV, actualV, "Expected %s to equal '%v', but got '%v'", dottedPath, expectedV, actualV)
			}
		})

	return output
}

func isMapType(v interface{}) bool {
	_, isMap := v.(Map)
	if isMap {
		return isMap
	}
	_, isStrictMap := v.(StrictMap)
	return isStrictMap
}
