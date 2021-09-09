// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package syncgateway

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func CreateTestMuxer() *http.ServeMux {
	mux := http.NewServeMux()

	mux.Handle("/_expvar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		input, _ := ioutil.ReadFile("../_meta/testdata/expvar.282c.json")
		_, err := w.Write(input)
		if err != nil {
			fmt.Println("error writing response on mock server")
		}
	}))

	return mux
}

func GetConfig(metricsets []string, host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "syncgateway",
		"metricsets": metricsets,
		"hosts":      []string{host},
	}
}
