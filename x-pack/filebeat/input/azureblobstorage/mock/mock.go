// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package mock

import (
	"net/http"
)

//nolint:errcheck // We can ignore as this is just for testing
func GCSServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(data))
		//path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")

		// if r.Method == http.MethodGet {
		// 	switch len(path) {
		// 	case 2:
		// 		if path[0] == "b" {
		// 			if buckets[path[1]] {
		// 				w.Write([]byte(fetchBucket[path[1]]))
		// 				return
		// 			}
		// 		} else if buckets[path[0]] && availableObjects[path[0]][path[1]] {
		// 			w.Write([]byte(objects[path[0]][path[1]]))
		// 			return
		// 		}
		// 	case 3:
		// 		if path[0] == "b" && path[2] == "o" {
		// 			if buckets[path[1]] {
		// 				w.Write([]byte(objectList[path[1]]))
		// 				return
		// 			}
		// 		} else if buckets[path[0]] {
		// 			objName := strings.Join(path[1:], "/")
		// 			if availableObjects[path[0]][objName] {
		// 				w.Write([]byte(objects[path[0]][objName]))
		// 				return
		// 			}
		// 		}
		// 	default:
		// 		w.WriteHeader(http.StatusNotFound)
		// 		return
		// 	}
		// }
		w.WriteHeader(http.StatusNotFound)
	})
}
