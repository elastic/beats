// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !integration
// +build !integration

package auto_ops_testing

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/require"

	libbeatversion "github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/utils"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/version"
)

var (
	versionRegexp = regexp.MustCompile(`((\d+)\.(\d+)\.(\d+))`)
)

// Simple callback to handle work against a matched globfile as part of a test
type GlobCallback func(t *testing.T, file string, version string, data []byte)

// The `server` will automatically have `defer server.Close()` closed
type SetupServerCallback func(t *testing.T, clusterInfo []byte, data []byte, version string) (server *httptest.Server)

// The config setup for the MetricSet
type SetupConfigCallback func(server *httptest.Server) (config map[string]interface{})

type GetTemplateCallback func(t *testing.T, names []string, ignoreNames []string) []byte

// Version number is in the filename as the third, forth, and fifth part.
// For example: `cluster_health.[reason].#.#.#.json` would return `#.#.#`.
func ExtractVersionFromFile(file string) (version string) {
	return versionRegexp.FindStringSubmatch(file)[0]
}

// Helper function to automatically use the server's URL and just the name as the metricset.
func UseNamedMetricSet(name string) SetupConfigCallback {
	return func(server *httptest.Server) map[string]interface{} {
		return map[string]interface{}{
			"module":     "autoops_es",
			"metricsets": []string{name},
			"hosts":      []string{server.URL},
		}
	}
}

// Setup a Server with the Cluster Info route set to fail (HTTP 401).
func SetupClusterInfoErrorServer(t *testing.T, _ []byte, _ []byte, _ string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/":
			w.WriteHeader(401)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"error":"Bad Auth"}`))
		default:
			t.Fatalf("Unrecognized request %v", r.RequestURI)
		}
	}))
}

// Setup a Server with the data route set to `dataRoute` that fails via HTTP 5xx.
func SetupDataErrorServer(dataRoute string) SetupServerCallback {
	return func(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(clusterInfo)
			case dataRoute:
				w.WriteHeader(500)
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"error":"Unexpected error"}`))
			default:
				t.Fatalf("Unrecognized request %v", r.RequestURI)
			}
		}))
	}
}

// Setup a Server with the data route set to `dataRoute`.
func SetupSuccessfulServer(dataRoute string) SetupServerCallback {
	return func(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(clusterInfo)
			case dataRoute:
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			default:
				t.Fatalf("Unrecognized request %v", r.RequestURI)
			}
		}))
	}
}

func SetupSuccessfulTemplateServer(path string, pathPrefix string, getTemplateResponse GetTemplateCallback) SetupServerCallback {
	return func(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(clusterInfo)
			case path:
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			default:
				if strings.HasPrefix(r.RequestURI, pathPrefix) {
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					w.Write(getTemplateResponse(t, strings.Split(r.RequestURI[len(pathPrefix):], ","), []string{}))

					return
				}

				t.Fatalf("Unrecognized request %v", r.RequestURI)
			}
		}))
	}
}

func SetupSuccessfulTemplateServerWithFailedRequests(path string, pathPrefix string, getTemplateResponse GetTemplateCallback, failedNames []string) SetupServerCallback {
	return func(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(clusterInfo)
			case path:
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			default:
				if strings.HasPrefix(r.RequestURI, pathPrefix) {
					templateNames := strings.Split(r.RequestURI[len(pathPrefix):], ",")

					if slices.ContainsFunc(failedNames, func(name string) bool {
						return slices.Contains(templateNames, name)
					}) {
						w.WriteHeader(500)
						w.Header().Set("Content-Type", "application/json")
						w.Write([]byte(`{"error":"Unexpected error"}`))

						return
					}

					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					w.Write(getTemplateResponse(t, templateNames, []string{}))

					return
				}

				t.Fatalf("Unrecognized request %v", r.RequestURI)
			}
		}))
	}
}

func SetupSuccessfulTemplateServerWithIgnoredTemplates(path string, pathPrefix string, getTemplateResponse GetTemplateCallback, ignoredNames []string) SetupServerCallback {
	return func(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(clusterInfo)
			case path:
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			default:
				if strings.HasPrefix(r.RequestURI, pathPrefix) {
					w.WriteHeader(200)
					w.Header().Set("Content-Type", "application/json")
					w.Write(getTemplateResponse(t, strings.Split(r.RequestURI[len(pathPrefix):], ","), ignoredNames))

					return
				}

				t.Fatalf("Unrecognized request %v", r.RequestURI)
			}
		}))
	}
}

func SetupTemplateErrorsServer(path string, pathPrefix string) SetupServerCallback {
	return func(t *testing.T, clusterInfo []byte, data []byte, _ string) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.RequestURI {
			case "/":
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(clusterInfo)
			case path:
				w.WriteHeader(200)
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			default:
				if strings.HasPrefix(r.RequestURI, pathPrefix) {
					w.WriteHeader(500)
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"error":"Unexpected error"}`))

					return
				}

				t.Fatalf("Unrecognized request %v", r.RequestURI)
			}
		}))
	}
}

// Tests a glob file pattern (must match at least one file) and extracts matching files, their version (from the name), and the data from the file.
// Each matching file runs as a separate test that includes the name.
func RunTestsForGlobFiles(t *testing.T, glob string, callback GlobCallback) {
	files, err := filepath.Glob(glob)
	require.NoError(t, err)
	// Makes sure glob matches at least 1 file
	require.True(t, len(files) > 0)

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			data, err := os.ReadFile(f)
			require.NoError(t, err)

			callback(t, f, ExtractVersionFromFile(f), data)
		})
	}
}

func CreateClusterInfo(clusterVersion string) utils.ClusterInfo {
	return utils.ClusterInfo{
		ClusterID:   "GZbSUUMQQI-A7UcGS6vCMa",
		ClusterName: "my-cluster",
		Version: utils.ClusterInfoVersion{
			Number:       version.MustNew(clusterVersion),
			Distribution: "rpm",
		},
	}
}

// Unravel `mapstr.M.GetValue` without an error response to make it easier to assert
func GetObjectValue(obj mapstr.M, key string) interface{} {
	exists, err := obj.HasKey(key)

	if !exists {
		return nil
	} else if err != nil {
		return err
	}

	value, err := obj.GetValue(key)

	if err != nil {
		return err
	}

	return value
}

// Function that uses GetObjectValue to retrieve the value and return it as a string
func GetObjectAsString(t *testing.T, obj mapstr.M, key string) string {
	value := GetObjectValue(obj, key)

	// Type assertion to convert the value to a string
	strValue, ok := value.(string)
	if !ok {
		// If the value is not a string, return an error with the actual type
		t.Fatalf("the value for the key '%s' is not a string, but is of type: %T", key, value)
		return ""
	}

	return strValue
}

// Unravel `mapstr.M.GetValue` and `mapstr.M.String` without an error response to make it easier to assert
func GetObjectAsJson(obj mapstr.M, key string) interface{} {
	exists, err := obj.HasKey(key)

	if err != nil {
		return err
	} else if !exists {
		return nil
	}

	value := mapstr.M{}

	err = obj.CopyFieldsTo(value, key)

	if err != nil {
		return err
	}

	return value.String()
}

// Get the event matching the name with the given key. If none match, the test will fail.
func GetEventByName(t *testing.T, events []mb.Event, key string, name string) mb.Event {
	for _, event := range events {
		if GetObjectValue(event.MetricSetFields, key) == name {
			return event
		}
	}

	t.Fatalf("No matching event for %v with %v", key, name)
	return mb.Event{} // impossible
}

// Get the events with the key defined. If none match, the test will fail.
func GetEventsWithField(t *testing.T, events []mb.Event, key string) []mb.Event {
	matchingEvents := []mb.Event{}

	for _, event := range events {
		if GetObjectValue(event.MetricSetFields, key) != nil {
			matchingEvents = append(matchingEvents, event)
		}
	}

	return matchingEvents
}

func CheckEvent(t *testing.T, event mb.Event, info utils.ClusterInfo) {
	require.Equal(t, info.ClusterID, GetObjectValue(event.ModuleFields, "cluster.id"))
	require.Equal(t, info.ClusterName, GetObjectValue(event.ModuleFields, "cluster.name"))
	require.Equal(t, info.Version.Number.String(), GetObjectValue(event.ModuleFields, "cluster.version"))

	require.Equal(t, "autoops_es", GetObjectValue(event.RootFields, "service.name"))
	require.Equal(t, libbeatversion.GetDefaultVersion(), GetObjectValue(event.RootFields, "metricbeatVersion"))
	require.Equal(t, libbeatversion.Commit(), GetObjectValue(event.RootFields, "commit"))
}

func CheckEventWithTransactionId(t *testing.T, event mb.Event, info utils.ClusterInfo, transactionId string) {
	CheckEvent(t, event, info)

	// matching transaction ID
	require.Equal(t, transactionId, GetObjectValue(event.ModuleFields, "transactionId"))
}

func CheckEventWithRandomTransactionId(t *testing.T, event mb.Event, info utils.ClusterInfo) {
	CheckEvent(t, event, info)

	// valid, random UUID
	_, err := uuid.FromString(GetObjectValue(event.ModuleFields, "transactionId").(string))
	require.NoError(t, err)
}

func CheckAllEventsUseSameTransactionId(t *testing.T, events []mb.Event) {
	if len(events) > 1 {
		transactionId := GetObjectValue(events[0].ModuleFields, "transactionId")

		for _, event := range events {
			require.Equal(t, transactionId, GetObjectValue(event.ModuleFields, "transactionId"))
		}
	}
}
