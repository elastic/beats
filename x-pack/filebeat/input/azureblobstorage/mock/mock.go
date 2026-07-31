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
	xmlType     = "application/xml"
)

//nolint:errcheck // We can ignore as response writer errors cannot be handled in this scenario
func AzureStorageServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		pathPrefix := r.URL.Query().Get("prefix")

		w.Header().Set(contentType, jsonType)
		if r.Method == http.MethodGet {
			switch len(path) {
			case 1:
				containerName := path[0]
				if !Containers[containerName] {
					// This case was implicitly handled by doing nothing.
					// Breaking is clear and maintains that behavior.
					break
				}

				// If a prefix is given but is not valid, return a 404 error.
				if pathPrefix != "" {
					containerBlobs := availableBlobs[containerName]
					if !hasKeyWithPrefix(containerBlobs, pathPrefix) {
						w.WriteHeader(http.StatusNotFound)
						w.Write([]byte("resource not found"))
						return
					}
				}

				w.Header().Set(contentType, xmlType)
				w.Write([]byte(filterListingByPrefix(fetchContainer[containerName], pathPrefix)))
				return
			case 2:
				if Containers[path[0]] && availableBlobs[path[0]][path[1]] {
					w.Write([]byte(blobs[path[0]][path[1]]))
					return
				}
			case 3:
				if Containers[path[0]] {
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

//nolint:errcheck // We can ignore as response writer errors cannot be handled in this scenario
func AzureStorageFileServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		if r.Method == http.MethodGet {
			switch len(path) {
			case 1:
				if fileContainers[path[0]] {
					w.Header().Set(contentType, xmlType)
					w.Write([]byte(fetchFilesContainer[path[0]]))
					return
				}
			case 2:
				if fileContainers[path[0]] && availableFileBlobs[path[0]][path[1]] {
					absPath, _ := filepath.Abs("testdata/" + path[1])
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
					case "txn1.csv":
						w.Header().Set(contentType, "text/csv")
					}
					//nolint:gosec // G705 false positive: test HTTP mock serving static fixture bytes
					w.Write(data)
					return
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

//nolint:errcheck // We can ignore as response writer errors cannot be handled in this scenario
func AzureFileServerNoContentType() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		if r.Method == http.MethodGet {
			switch len(path) {
			case 1:
				if fileContainers[path[0]] {
					w.Write([]byte(fetchFilesContainer[path[0]]))
					return
				}
			case 2:
				if availableFileBlobs[path[0]][path[1]] {
					absPath, _ := filepath.Abs("testdata/" + path[1])
					data, _ := os.ReadFile(absPath)
					//nolint:gosec // G705 false positive: test HTTP mock serving static fixture bytes
					w.Write(data)
					return
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

//nolint:errcheck // We can ignore as response writer errors cannot be handled in this scenario
func AzureConcurrencyServer() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
		w.Header().Set(contentType, jsonType)
		if r.Method == http.MethodGet {
			switch len(path) {
			case 1:
				if path[0] == ConcurrencyContainer {
					w.Header().Set(contentType, xmlType)
					w.Write(generateMetadata())
					return
				}
			case 2:
				w.Write(generateRandomBlob())
				return
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("resource not found"))
	})
}

// hasKeyWithPrefix checks if any key in a map starts with the given prefix.
func hasKeyWithPrefix(data map[string]bool, prefix string) bool {
	for key := range data {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

// filterListingByPrefix returns a container listing XML containing only blobs
// whose names match prefix. An empty prefix returns the full listing.
func filterListingByPrefix(listing, prefix string) string {
	if prefix == "" {
		return listing
	}

	var out strings.Builder
	rest := listing
	for {
		start := strings.Index(rest, "<Blob>")
		if start == -1 {
			out.WriteString(rest)
			break
		}
		out.WriteString(rest[:start])
		rest = rest[start:]
		end := strings.Index(rest, "</Blob>")
		if end == -1 {
			out.WriteString(rest)
			break
		}
		blob := rest[:end+len("</Blob>")]
		rest = rest[end+len("</Blob>"):]

		nameStart := strings.Index(blob, "<Name>") + len("<Name>")
		nameEnd := strings.Index(blob, "</Name>")
		if nameStart < len("<Name>") || nameEnd == -1 {
			continue
		}
		if strings.HasPrefix(blob[nameStart:nameEnd], prefix) {
			out.WriteString(blob)
		}
	}
	return out.String()
}
