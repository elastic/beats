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

//go:build !windows
// +build !windows

package decode_xml_wineventlog

import (
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/winlogbeat/sys/winevent"
)

type nonWinDecoder struct{}

func newDecoder() decoder {
	return nonWinDecoder{}
}

func (nonWinDecoder) decode(data []byte) (common.MapStr, common.MapStr, error) {
	evt, err := winevent.UnmarshalXML(data)
	if err != nil {
		return nil, nil, err
	}
	winevent.EnrichRawValuesWithNames(nil, &evt)
	win, ecs := fields(evt)
	return win, ecs, nil
}
