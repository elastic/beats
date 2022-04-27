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

package stdfields

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestLegacyServiceNameConfig(t *testing.T) {
	srvName := "myService"

	configBase := func() common.MapStr {
		return common.MapStr{
			"type":     "http",
			"id":       "myId",
			"schedule": "@every 1s",
		}
	}

	legacyOnly := configBase()
	legacyOnly["service_name"] = srvName

	newOnly := configBase()
	newOnly["service"] = common.MapStr{"name": srvName}

	mix := configBase()
	mix["service"] = common.MapStr{"name": srvName}
	mix["service_name"] = "ignoreMe"

	confMaps := []common.MapStr{
		legacyOnly,
		newOnly,
		mix,
	}

	for _, cm := range confMaps {
		t.Run(fmt.Sprintf("given config map %#v", cm), func(t *testing.T) {
			c, err := conf.NewConfigFrom(cm)
			require.NoError(t, err)
			f, err := ConfigToStdMonitorFields(c)
			require.Equal(t, srvName, f.Service.Name)
		})
	}

}
