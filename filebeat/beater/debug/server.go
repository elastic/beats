// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
//
// This file was contributed to by generative AI

package debug

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	bolt "go.etcd.io/bbolt"

	"github.com/elastic/beats/v7/libbeat/statestore/backend/bbolt"
	"github.com/elastic/elastic-agent-libs/logp"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
	dataBucketName  = "data"
)

type Server struct {
	logger    *logp.Logger
	port      int
	registry  *bbolt.Registry
	server    *http.Server
	storeName string
}

type PageResponse struct {
	Keys       []KeyValue `json:"keys"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

type KeyValue struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
	Error string          `json:"error,omitempty"`
}

func NewServer(logger *logp.Logger, port int, registry *bbolt.Registry, storeName string) (*Server, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}
	if registry == nil {
		return nil, fmt.Errorf("registry is nil")
	}

	s := &Server{
		logger:    logger.Named("debug"),
		port:      port,
		registry:  registry,
		storeName: storeName,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleUI)
	mux.HandleFunc("/api/keys", s.handleKeys)

	s.server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Errorf("Debug server error: %v", err)
		}
	}()

	s.logger.Infof("Debug server started on port %d", port)
	return s, nil
}

func (s *Server) Stop() error {
	if s.server == nil {
		return nil
	}
	s.logger.Info("Stopping debug server")
	return s.server.Close()
}

func (s *Server) handleUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlTemplate))
}

func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	page, err := parseIntParam(r, "page", 1)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid page parameter: %v", err), http.StatusBadRequest)
		return
	}

	pageSize, err := parseIntParam(r, "page_size", defaultPageSize)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid page_size parameter: %v", err), http.StatusBadRequest)
		return
	}

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if page < 1 {
		page = 1
	}

	db := s.registry.GetDB(s.storeName)
	if db == nil {
		http.Error(w, "registry store not found or closed", http.StatusNotFound)
		return
	}

	var response PageResponse
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(dataBucketName))
		if bucket == nil {
			return nil
		}

		// Count total keys
		stats := bucket.Stats()
		total := stats.KeyN

		// Calculate pagination
		totalPages := (total + pageSize - 1) / pageSize
		if totalPages == 0 {
			totalPages = 1
		}
		if page > totalPages {
			page = totalPages
		}

		response.Total = total
		response.Page = page
		response.PageSize = pageSize
		response.TotalPages = totalPages

		// Skip to offset
		offset := (page - 1) * pageSize
		cursor := bucket.Cursor()
		count := 0
		skipped := 0

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if skipped < offset {
				skipped++
				continue
			}
			if count >= pageSize {
				break
			}

			kv := KeyValue{
				Key: string(k),
			}

			// Try to parse as JSON for pretty printing
			var raw json.RawMessage
			if err := json.Unmarshal(v, &raw); err != nil {
				kv.Error = fmt.Sprintf("invalid JSON: %v", err)
				kv.Value = json.RawMessage(v)
			} else {
				kv.Value = raw
			}

			response.Keys = append(response.Keys, kv)
			count++
		}

		return nil
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func parseIntParam(r *http.Request, name string, defaultValue int) (int, error) {
	val := r.URL.Query().Get(name)
	if val == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(val)
}
