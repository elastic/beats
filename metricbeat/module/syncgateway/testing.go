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
