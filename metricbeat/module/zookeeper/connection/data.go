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

package connection

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"

	"github.com/elastic/beats/v7/metricbeat/mb"

	"github.com/elastic/beats/v7/libbeat/common"
)

var capturer = regexp.MustCompile(`/(?P<ip>.*):(?P<port>\d+)\[(?P<interest_ops>\d*)]\(queued=(?P<queued>\d*),recved=(?P<received>\d*),sent=(?P<sent>\d*)\)`)

func (m *MetricSet) parseCons(i io.Reader) []mb.Event {
	scanner := bufio.NewScanner(i)

	result := make([]mb.Event, 0)

	for scanner.Scan() {
		metricsetFields := common.MapStr{}
		rootFields := common.MapStr{}
		line := scanner.Text()

		oneParsingIsCorrect := false
		keyMap, err := lineToMap(line)
		if err != nil {
			m.Logger().Errorf("Error while parsing zookeeper 'cons' command %s", err.Error())
			continue
		}

		for k, v := range keyMap {
			if k == "ip" {
				if _, err := rootFields.Put("client.ip", v); err != nil {
					m.Logger().Debugf("%v. Error placing key 'ip' on event", err)
				} else {
					oneParsingIsCorrect = true
				}
			} else if k == "port" {
				m.checkRegexAndSetInt(rootFields, v, "client.port", &oneParsingIsCorrect)
			} else {
				m.checkRegexAndSetInt(metricsetFields, v, k, &oneParsingIsCorrect)
			}
		}

		if oneParsingIsCorrect {
			result = append(result, mb.Event{MetricSetFields: metricsetFields, RootFields: rootFields})
		} else {
			m.Logger().Debug("no field from incoming string '%s' could be parsed", line)
		}
	}

	return result
}

func lineToMap(line string) (map[string]string, error) {
	capturedPatterns := capturer.FindStringSubmatch(line)
	if len(capturedPatterns) < 1 {
		//Nothing captured
		return nil, fmt.Errorf("no data captured in '%s'", line)
	}

	keyMap := make(map[string]string)
	for i, name := range capturer.SubexpNames() {
		if i != 0 && name != "" {
			keyMap[name] = capturedPatterns[i]
		}
	}

	return keyMap, nil
}

func (m *MetricSet) checkRegexAndSetInt(output common.MapStr, capturedData string, key string, correct *bool) {
	if capturedData != "" {
		n, err := strconv.ParseInt(capturedData, 10, 64)
		if err != nil {
			m.Logger().Errorf("parse error: %v. Cannot convert string to it", err)
			return
		}
		if _, err = output.Put(key, n); err != nil {
			m.Logger().Errorf("parse error: %v. Error putting key '%s' on event", err, key)
			return
		}
		*correct = true
	} else {
		m.Logger().Errorf("parse error: empty data for key '%s'", key)
	}
}
