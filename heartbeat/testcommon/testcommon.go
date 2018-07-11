package testcommon

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

var IsDuration = func(t *testing.T, actual interface{}) {
	converted, ok := actual.(time.Duration)
	assert.True(t, ok)
	assert.True(t, converted >= 0)
}

var IsNil = func(t *testing.T, actual interface{}) {
	assert.Nil(t, actual)
}

var IsString = func(t *testing.T, actual interface{}) {
	_, ok := actual.(string)
	assert.True(t, ok)
}

type MapLeafTest = func(t *testing.T, actual interface{})

type MapCheckDef common.MapStr

func DeepMapStrCheck(t *testing.T, expected MapCheckDef, actual common.MapStr) {
	deepMapStrCheckPath(t, expected, actual, actual, []string{})
}

func deepMapStrCheckPath(t *testing.T, expected MapCheckDef, actual common.MapStr, rootActual common.MapStr, path []string) {
	fmt.Printf("A: %v\n", actual)
	for expectedK, expectedV := range expected {
		expectedV := expectedV // Bind locally for subsequent t.Run calls
		keyPath := strings.Join(path, ".") + "." + expectedK
		mapLeafTest, ok := expectedV.(MapLeafTest)
		actualV, err := actual.GetValue(expectedK)
		assert.Nil(t, err, fmt.Sprintf("Expectation exists for %s', but the given MapStr does not have it: %v", keyPath, rootActual))
		if ok {
			t.Run(fmt.Sprintf("map path|customMatch(%s)", keyPath), func(t *testing.T) { mapLeafTest(t, actualV) })
		} else if actualVMapStr, nestedOK := actualV.(common.MapStr); nestedOK {
			expectedVMapStr, ok := expectedV.(MapCheckDef)
			if ok != true {
				t.FailNow()
			}
			assert.True(t, ok)
			deepMapStrCheckPath(t, expectedVMapStr, actualVMapStr, rootActual, append(path, expectedK))
		} else {
			// assert exact equality otherwise
			t.Run(fmt.Sprintf("map path|%s=>%v", keyPath, expectedV), func(t *testing.T) {
				assert.Equal(t, expectedV, actualV, "Expected %s to equal '%v', but got '%v'", keyPath, expectedV, actualV)
			})
		}
	}
}
