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
	"context"
	"fmt"
	"time"

	wmi "github.com/microsoft/wmi/pkg/wmiinstance"

	"github.com/elastic/elastic-agent-libs/logp"
)

// Wrapper of the session.QueryInstances function that execute a query for at most a timeout
// after which we stop actively waiting.
// Note that the underlying query will continue to run, until the query completes or the WMI Arbitrator stops the query
// https://learn.microsoft.com/en-us/troubleshoot/windows-server/system-management-components/new-wmi-arbitrator-behavior-in-windows-server
func ExecuteGuardedQueryInstances(session *wmi.WmiSession, query string, timeout time.Duration) ([]*wmi.WmiInstance, error) {
	var rows []*wmi.WmiInstance
	var err error
	done := make(chan error)
	timedout := false

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		start_time := time.Now()
		rows, err = session.QueryInstances(query)
		if !timedout {
			done <- err
		} else {
			timeSince := time.Since(start_time)
			baseMessage := fmt.Sprintf("The timed out query '%s' terminated after %s", query, timeSince)
			// We eventually fetched the documents, let us free them
			if err == nil {
				logp.Warn("%s successfully. The result will be ignored", baseMessage)
				wmi.CloseAllInstances(rows)
			} else {
				logp.Warn("%s with an error %v", baseMessage, err)
			}
		}
	}()

	select {
	case <-ctx.Done():
		err = fmt.Errorf("the execution of the query'%s' exceeded the threshold of %s", query, timeout)
		timedout = true
		close(done)
	case <-done:
		// Query completed in time either successfully or with an error
	}
	return rows, err
}
