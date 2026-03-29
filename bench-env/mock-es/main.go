// mock-es is a minimal Elasticsearch mock that accepts and discards bulk data.
// It responds just enough to satisfy filebeat's startup handshake and bulk indexing.
package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

var docsIngested atomic.Int64

func main() {
	addr := ":9200"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	mux := http.NewServeMux()

	// Stats endpoint for retrieving doc count
	mux.HandleFunc("GET /_mock/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"docs_ingested":%d}`, docsIngested.Load())
	})

	// Catch-all: route everything through a single handler that inspects
	// the path to decide what to return. This avoids Go 1.22+ ServeMux
	// pattern conflicts between method-specific and wildcard routes.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.Header().Set("Content-Type", "application/json")

		path := r.URL.Path

		// Bulk ingest — accept and discard
		if r.Method == "POST" && (path == "/_bulk" || len(path) > 6 && path[len(path)-6:] == "/_bulk") {
			bulkHandler(w, r)
			return
		}

		// Drain any request body
		io.Copy(io.Discard, r.Body)

		switch {
		case r.Method == "GET" && path == "/":
			// Cluster info
			fmt.Fprint(w, `{"name":"mock","cluster_name":"mock","cluster_uuid":"mock","version":{"number":"8.17.0","build_flavor":"default","build_type":"docker","lucene_version":"9.12.0","minimum_wire_compatibility_version":"7.17.0","minimum_index_compatibility_version":"7.0.0"},"tagline":"You Know, for Search"}`)
		case r.Method == "GET" && path == "/_license":
			fmt.Fprint(w, `{"license":{"uid":"mock","type":"trial","status":"active","expiry_date_in_millis":4102444800000}}`)
		case strings.HasPrefix(path, "/_index_template") && r.Method == "GET":
			fmt.Fprint(w, `{"index_templates":[]}`)
		case strings.HasPrefix(path, "/_component_template") && r.Method == "GET":
			fmt.Fprint(w, `{"component_templates":[]}`)
		case strings.HasPrefix(path, "/_ilm/policy") && r.Method == "GET":
			fmt.Fprint(w, `{}`)
		default:
			fmt.Fprint(w, `{"acknowledged":true}`)
		}
	})

	fmt.Fprintf(os.Stderr, "mock-es listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "mock-es: %v\n", err)
		os.Exit(1)
	}
}

func bulkHandler(w http.ResponseWriter, r *http.Request) {
	// Read body and count action lines to determine document count.
	// Decompress if gzipped.
	var bodyReader io.Reader = r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(r.Body)
		if err == nil {
			defer gz.Close()
			bodyReader = gz
		}
	}

	// Bulk format is pairs of lines: action\n source\n
	// Action lines start with {"create", {"index", or {"update".
	body, _ := io.ReadAll(bodyReader)
	docs := 0
	for i := 0; i < len(body); {
		// Find start of line
		if body[i] == '{' && i+8 < len(body) {
			// Check if this is an action line (starts with {"create, {"index, or {"update)
			prefix := string(body[i : i+8])
			if prefix == `{"create` || prefix == `{"index"` || prefix == `{"update` || prefix == `{"delete` {
				docs++
			}
		}
		// Skip to next line
		for i < len(body) && body[i] != '\n' {
			i++
		}
		i++ // skip the \n
	}
	if docs < 1 && len(body) > 0 {
		docs = 1
	}
	docsIngested.Add(int64(docs))

	// Build a response with one item per document.
	w.Header().Set("X-Elastic-Product", "Elasticsearch")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"took":1,"errors":false,"items":[`))
	for i := 0; i < docs; i++ {
		if i > 0 {
			w.Write([]byte(","))
		}
		w.Write([]byte(`{"create":{"_index":"mock","_id":"mock","_version":1,"result":"created","status":201}}`))
	}
	w.Write([]byte("]}"))
}
