// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mock

import (
	"net/http"
	"strings"
)

//nolint:errcheck // We can ignore as response writer errors cannot be handled in this scenario
func AzureStorageServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			switch len(path) {
			case 1:
				if containers[path[0]] {
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(fetchContainer[path[0]]))
					return
				}
			case 2:
				if containers[path[0]] && availableBlobs[path[0]][path[1]] {
					w.Write([]byte(blobs[path[0]][path[1]]))
					return
				}
			case 3:
				if containers[path[0]] {
					objName := strings.Join(path[1:], "/")
					if availableBlobs[path[0]][objName] {
						w.Write([]byte(blobs[path[0]][objName]))
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
