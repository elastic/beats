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

package decode_xml

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
)

func wineventlogFields(evt winevent.Event) (common.MapStr, common.MapStr) {
	win := evt.Fields()

	ecs := common.MapStr{}

	ecs.Put("event.kind", "event")
	ecs.Put("event.code", fmt.Sprint(evt.EventIdentifier.ID))
	ecs.Put("event.provider", evt.Provider.Name)
	winevent.AddOptional(ecs, "event.action", evt.Task)
	winevent.AddOptional(ecs, "host.name", evt.Computer)
	winevent.AddOptional(ecs, "event.outcome", getValue(win, "outcome"))
	winevent.AddOptional(ecs, "log.level", getValue(win, "level"))
	winevent.AddOptional(ecs, "message", getValue(win, "message"))
	winevent.AddOptional(ecs, "error.code", getValue(win, "error.code"))
	winevent.AddOptional(ecs, "error.message", getValue(win, "error.message"))

	return win, ecs
}

func getValue(m common.MapStr, key string) interface{} {
	v, _ := m.GetValue(key)
	return v
}
