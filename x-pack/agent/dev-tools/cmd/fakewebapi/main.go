// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/fleetapi"
)

var (
	host   string
	apiKey string
	mutex  sync.Mutex

	pathCheckin     = regexp.MustCompile(`^/api/fleet/agents/(.+)/checkin`)
	checkinResponse = response{Actions: make([]action, 0), Success: true}
)

type response struct {
	Actions []action `json:"actions"`
	Success bool     `json:"success"`
}

type action interface{}

func init() {
	flag.StringVar(&apiKey, "apikey", "abc123", "API Key to authenticate")
	flag.StringVar(&host, "host", "localhost:8080", "The IP and port to use for the webserver")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/fleet/agents/enroll", handlerEnroll)
	mux.HandleFunc("/admin/actions", handlerAction)
	mux.HandleFunc("/", handlerRoot)

	log.Printf("Starting webserver and listening on %s", host)

	listener, err := net.Listen("tcp", host)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	http.Serve(listener, mux)
}

func handlerRoot(w http.ResponseWriter, r *http.Request) {
	if pathCheckin.MatchString(r.URL.Path) {
		authHandler(handlerCheckin, apiKey)(w, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{ "message": "Hello!"}`))
	log.Println("Root hello!")
	log.Println("Path: ", r.URL.Path)
}

func handlerEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	response := &fleetapi.EnrollResponse{
		Action:  "created",
		Success: true,
		Item: fleetapi.EnrollItemResponse{
			ID:                   "a4937110-e53e-11e9-934f-47a8e38a522c",
			Active:               true,
			PolicyID:             "default",
			Type:                 fleetapi.PermanentEnroll,
			EnrolledAt:           time.Now(),
			UserProvidedMetadata: make(map[string]interface{}),
			LocalMetadata:        make(map[string]interface{}),
			AccessAPIKey:         apiKey,
		},
	}

	b, err := json.Marshal(response)
	if err != nil {
		log.Printf("failed to enroll: %+v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(b)
	log.Println("Enroll response:", string(b))
}

func handlerCheckin(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	b, err := json.Marshal(checkinResponse)
	if err != nil {
		log.Printf("Failed to checkin, error: %+v", err)
		http.Error(w, "Internal Server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(b)
	log.Println("Checkin response: ", string(b))
}

func handlerAction(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()
	if r.Method != "POST" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	resp := response{}

	var buf bytes.Buffer
	tee := io.TeeReader(r.Body, &buf)

	c, err := ioutil.ReadAll(tee)
	if err != nil {
		log.Printf("Fails to update the actions")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	decoder := json.NewDecoder(&buf)
	err = decoder.Decode(&resp)
	if err != nil {
		log.Printf("Fails to update the actions")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	checkinResponse = resp
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{ "success": true }`))
	log.Println("Action request: ", string(c))
}

func authHandler(handler http.HandlerFunc, apiKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const key = "Authorization"
		const prefix = "ApiKey "

		v := strings.TrimPrefix(r.Header.Get(key), prefix)
		if v != apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}
}
