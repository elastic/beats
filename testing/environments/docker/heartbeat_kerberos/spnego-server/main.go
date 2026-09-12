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

// Command spnego-server is a tiny HTTP server protected by SPNEGO/Negotiate
// (Kerberos). It is used only by the Heartbeat HTTP monitor Kerberos
// integration test: authenticated requests receive a 200, everything else is
// handled by the gokrb5 SPNEGO handler (401 with WWW-Authenticate: Negotiate).
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/elastic/gokrb5/v8/keytab"
	"github.com/elastic/gokrb5/v8/service"
	"github.com/elastic/gokrb5/v8/spnego"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	ktPath := flag.String("keytab", "/etc/http.keytab", "path to the service keytab")
	flag.Parse()

	kt, err := keytab.Load(*ktPath)
	if err != nil {
		log.Fatalf("loading keytab %q: %v", *ktPath, err)
	}

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("authenticated\n"))
	})

	handler := spnego.SPNEGOKRB5Authenticate(inner, kt, service.Logger(log.Default()))

	log.Printf("SPNEGO server listening on %s", *addr)
	srv := &http.Server{Addr: *addr, Handler: handler}
	log.Fatal(srv.ListenAndServe())
}
