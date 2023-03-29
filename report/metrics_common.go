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

package report

import (
	"os"
	"os/user"
	"strconv"
	"time"

	"github.com/gofrs/uuid"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var (
	ephemeralID    uuid.UUID
	processMetrics *monitoring.Registry
	startTime      time.Time
)

func init() {
	startTime = time.Now()
	processMetrics = monitoring.Default.NewRegistry("beat")

	var err error
	ephemeralID, err = uuid.NewV4()
	if err != nil {
		logp.Err("Error while generating ephemeral ID for Beat")
	}
}

// EphemeralID returns generated EphemeralID
func EphemeralID() uuid.UUID {
	return ephemeralID
}

// SetupInfoUserMetrics adds user data to the `info` registry component
// this is performed async, as on windows user lookup can take up to a minute.
func SetupInfoUserMetrics() {
	infoRegistry := monitoring.GetNamespace("info").GetRegistry()
	go func() {
		if u, err := user.Current(); err != nil {
			// This usually happens if the user UID does not exist in /etc/passwd. It might be the case on K8S
			// if the user set securityContext.runAsUser to an arbitrary value.
			monitoring.NewString(infoRegistry, "uid").Set(strconv.Itoa(os.Getuid()))
			monitoring.NewString(infoRegistry, "gid").Set(strconv.Itoa(os.Getgid()))
		} else {
			monitoring.NewString(infoRegistry, "username").Set(u.Username)
			monitoring.NewString(infoRegistry, "uid").Set(u.Uid)
			monitoring.NewString(infoRegistry, "gid").Set(u.Gid)
		}
	}()
}
