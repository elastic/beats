// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

import (
	"net/http"
	"strings"
)

const (
	bucket = "gcs-test-new"
)

//nolint:errcheck // We can ignore as this is just for testing
func GCSServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		if r.Method == http.MethodGet {
			switch len(path) {
			case 2:
				if path[0] == "b" {
					if availableBuckets[path[1]] != nil {
						w.Write([]byte(availableBuckets[path[1]].(string)))
						return
					}
				} else if path[0] == bucket && availableObjects[path[1]] {
					w.Write([]byte(objects[path[1]].(string)))
					return
				}
			case 3:
				if path[0] == "b" && path[2] == "o" {
					if availableBuckets[path[1]] != nil {
						w.Write([]byte(objectList[path[1]].(string)))
						return
					}
				} else if path[0] == bucket {
					objName := strings.Join(path[1:], "/")
					if availableObjects[objName] {
						w.Write([]byte(objects[objName].(string)))
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
