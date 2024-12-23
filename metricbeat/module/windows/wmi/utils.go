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

package wmi

import (
	"fmt"
	"time"

	wmi "github.com/microsoft/wmi/pkg/wmiinstance"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Wrapper of the session.QueryInstances function that execute a query for at most a timeout
// Note that the underlying query will continue run
func ExecuteGuardedQueryInstances(session *wmi.WmiSession, query string, timeout time.Duration) ([]*wmi.WmiInstance, error) {
	var rows []*wmi.WmiInstance
	var err error
	done := make(chan bool)

	go func() {
		rows, err = session.QueryInstances(query)
		if err != nil {
			logp.Warn("Could not execute query %v", err)
		}
		done <- true
	}()

	select {
	case <-done:
		logp.Info("Query completed in time")
	case <-time.After(timeout):
		err = fmt.Errorf("query '%s' exceeded the timeout of %d", query, timeout)
		logp.Error(err)
	}

	return rows, err
}
