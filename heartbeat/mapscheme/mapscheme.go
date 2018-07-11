package mapscheme

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"fmt"

	"io"
	"net/http"

	"net/http/httptest"
	"net/url"
	"strconv"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

var HelloWorldBody = "hello, world!"

var HelloWorldHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, HelloWorldBody)
})

var BadGatewayBody = "Bad Gateway"

var BadGatewayHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadGateway)
	io.WriteString(w, BadGatewayBody)
})

var ExactlyEqual = func(expected interface{}) func(t *testing.T, actual interface{}) {
	return func(t *testing.T, actual interface{}) {
		assert.Equal(t, expected, actual)
	}
}

func ServerPort(server *httptest.Server) (uint16, error) {
	u, err := url.Parse(server.URL)
	if err != nil {
		return 0, err
	}
	p, err := strconv.Atoi(u.Port())
	if err != nil {
		return 0, err
	}
	return uint16(p), nil
}

// Functions for testing maps in complex ways

func MonitorChecks(id string, ip string, scheme string, status string) MapCheckDef {
	return MapCheckDef{
		"monitor": MapCheckDef{
			"duration.us": IsDuration,
			"id":          id,
			"ip":          ip,
			"scheme":      scheme,
			"status":      status,
		},
	}
}

func TcpChecks(port uint16) MapCheckDef {
	return MapCheckDef{
		"tcp": MapCheckDef{
			"port":           port,
			"rtt.connect.us": IsDuration,
		},
	}
}

var IsDuration = func(t *testing.T, _ bool, actual interface{}) {
	converted, ok := actual.(time.Duration)
	assert.True(t, ok)
	assert.True(t, converted >= 0)
}

var IsNil = func(t *testing.T, _ bool, actual interface{}) {
	assert.Nil(t, actual)
}

var IsString = func(t *testing.T, _ bool, actual interface{}) {
	_, ok := actual.(string)
	assert.True(t, ok)
}

type ValueValidator = func(t *testing.T, keyExists bool, actual interface{})

type MapCheckDef map[string]interface{}

type Validator func(*testing.T, common.MapStr) *ValidationResult

type ValidationResult struct {
	expectedFields   map[string]struct{}
	unexpectedFields map[string]struct{}
}

func combine(results []*ValidationResult) *ValidationResult {
	// Use sets to de-dupe these
	output := ValidationResult{}

	for _, res := range results {
		for k, _ := range res.expectedFields {
			output.expectedFields[k] = struct{}{}
		}
	}

	for _, res := range results {
		for k := range res.unexpectedFields {
			// Unexpected fields are those that are in no expectedFields
			// for any test
			if _, ok := output.expectedFields[k]; ok == false {
				output.unexpectedFields[k] = struct{}{}
			}
		}
	}

	return &output
}

func Compose(validators ...Validator) Validator {
	return func(t *testing.T, actual common.MapStr) *ValidationResult {
		results := make([]*ValidationResult, len(validators))
		for _, validator := range validators {
			result := validator(t, actual)
			results = append(results, result)
		}
		return combine(results)
	}
}

func Strict(validator Validator) Validator {
	return func(t *testing.T, actual common.MapStr) *ValidationResult {
		res := validator(t, actual)

		return res
	}
}

func Scheme(expected MapCheckDef) Validator {
	return func(t *testing.T, actual common.MapStr) *ValidationResult {
		return Validate(t, expected, actual)
	}
}

func Validate(t *testing.T, expected MapCheckDef, actual common.MapStr) *ValidationResult {
	return validateInternal(t, expected, actual, actual, []string{})
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
		wo(k, v, m, root, path, strings.Join(path, "."))

		// Walk nested maps
		if mapV, ok := v.(common.MapStr); ok {
			newPath := make([]string, len(path)+1)
			copy(newPath, path)
			newPath[len(path)] = k // Append the key
			walkFull(mapV, root, newPath, wo)
		}
	}
}

func walkValidate(t *testing.T, expected MapCheckDef, actual common.MapStr) (output *ValidationResult) {
	walk(
		common.MapStr(expected),
		func(expectedK string,
			expectedV interface{},
			currentMap common.MapStr,
			rootMap common.MapStr,
			path []string,
			dottedPath string) {
			actualHasKey, err := actual.HasKey(dottedPath)
			require.Nil(t, err)
			actualV, err := actual.GetValue(dottedPath)
			require.Nil(t, err)

			vv, isVV := expectedV.(ValueValidator)
			if isVV {
				t.Run(fmt.Sprintf("map path|customMatch(%s)", dottedPath), func(t *testing.T) { vv(t, actualHasKey, actualV) })
			} else { // Assert exact equality
				t.Run(fmt.Sprintf("map path|%s=>%v", dottedPath, expectedV), func(t *testing.T) {
					assert.Equal(t, expectedV, actualV, "Expected %s to equal '%v', but got '%v'", dottedPath, expectedV, actualV)
				})
			}

		})
}

func validateInternal(t *testing.T, expected MapCheckDef, actual common.MapStr, rootActual common.MapStr, path []string) (output *ValidationResult) {
	fmt.Printf("A: %v\n", actual)

	for expectedK, expectedV := range expected {
		expectedV := expectedV // Bind locally for subsequent t.Run calls
		keyPath := strings.Join(path, ".") + "." + expectedK
		mapLeafTest, ok := expectedV.(ValueValidator)

		hasKey, err := actual.HasKey(expectedK)
		require.Nil(t, err)
		require.True(t, hasKey, fmt.Sprintf("Expectation exists for %s', but the given MapStr does not have it: %v", keyPath, rootActual))
		if hasKey {
			output.expectedFields[keyPath] = struct{}{}
		}
		actualV, _ := actual.GetValue(expectedK)

		if ok {
			t.Run(fmt.Sprintf("map path|customMatch(%s)", keyPath), func(t *testing.T) { mapLeafTest(t, actualV) })
		} else if actualVMapStr, nestedOK := actualV.(common.MapStr); nestedOK {
			expectedVMapStr, ok := expectedV.(MapCheckDef)
			if ok != true {
				t.FailNow()
			}
			assert.True(t, ok)
			validateInternal(t, expectedVMapStr, actualVMapStr, rootActual, append(path, expectedK))
		} else {
			// assert exact equality otherwise
			t.Run(fmt.Sprintf("map path|%s=>%v", keyPath, expectedV), func(t *testing.T) {
				assert.Equal(t, expectedV, actualV, "Expected %s to equal '%v', but got '%v'", keyPath, expectedV, actualV)
			})
		}
	}

	return output
}
