// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	contentType = "Content-Type"
	jsonType    = "application/json"
)

//nolint:errcheck // We can ignore as this is just for testing
func GCSServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		if r.Method == http.MethodGet {
			switch len(path) {
			case 2:
				if path[0] == "b" {
					if buckets[path[1]] {
						w.Write([]byte(fetchBucket[path[1]]))
						return
					}
				} else if buckets[path[0]] && availableObjects[path[0]][path[1]] {
					w.Write([]byte(objects[path[0]][path[1]]))
					return
				}
			case 3:
				if path[0] == "b" && path[2] == "o" {
					if buckets[path[1]] {
						w.Write([]byte(objectList[path[1]]))
						return
					}
				} else if buckets[path[0]] {
					objName := strings.Join(path[1:], "/")
					if availableObjects[path[0]][objName] {
						w.Write([]byte(objects[path[0]][objName]))
						return
					}
				}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})
}

//nolint:errcheck // We can ignore as response writer errors cannot be handled in this scenario
func GCSFileServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		if r.Method == http.MethodGet {
			switch len(path) {
			case 2:
				if path[0] == "b" {
					if fileBuckets[path[1]] {
						w.Write([]byte(fetchFileBuckets[path[1]]))
						return
					}
				} else if fileBuckets[path[0]] && availableFileObjects[path[0]][path[1]] {
					absPath, _ := filepath.Abs("mock/testdata/" + path[1])
					data, _ := os.ReadFile(absPath)
					switch path[1] {
					case "multiline.json":
						w.Header().Set(contentType, "application/octet-stream")
					case "multiline.json.gz":
						w.Header().Set(contentType, jsonType)
					case "log.json", "events-array.json":
						w.Header().Set(contentType, jsonType)
					case "log.ndjson":
						w.Header().Set(contentType, "application/x-ndjson")
					}
					w.Write(data)
					return
				}
			case 3:
				if path[0] == "b" && path[2] == "o" {
					if fileBuckets[path[1]] {
						w.Write([]byte(objectFileList[path[1]]))
						return
					}
				}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("resource not found"))
	})
}
