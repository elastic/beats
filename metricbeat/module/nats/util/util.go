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

package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// convertUptimeToSeconds converts uptime from formatted string to seconds
// input: "1y20d22h3m30s", output: 33343410
func convertUptimeToSeconds(uptime string) (seconds int64, err error) {

	var split []string
	var years, days, hours, minutes, secs int64
	if strings.Contains(uptime, "y") {
		split = strings.Split(uptime, "y")
		uptime = split[1]
		years, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid years format in json data")
			return
		}
		seconds += years * 31536000
	}

	if strings.Contains(uptime, "d") {
		split = strings.Split(uptime, "d")
		uptime = split[1]
		days, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid days format in json data")
			return
		}
		seconds += days * 86400
	}

	if strings.Contains(uptime, "h") {
		split = strings.Split(uptime, "h")
		uptime = split[1]
		hours, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid hours format in json data")
			return
		}
		seconds += hours * 3600
	}

	if strings.Contains(uptime, "m") {
		split = strings.Split(uptime, "m")
		uptime = split[1]
		minutes, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid minutes format in json data")
			return
		}
		seconds += minutes * 60
	}

	if strings.Contains(uptime, "s") {
		split = strings.Split(uptime, "s")
		uptime = split[1]
		secs, err = strconv.ParseInt(split[0], 10, 64)
		if err != nil {
			err = errors.Wrap(err, "invalid seconds format in json data")
			return
		}
		seconds += secs
	}
	return
}

// UpdateDuration updates a duration in a mapstr.M from formatted string to seconds
func UpdateDuration(event mapstr.M, key string) error {
	item, err := event.GetValue(key)
	if err != nil {
		return nil
	}
	itemConverted, err := convertUptimeToSeconds(item.(string))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failure converting %v key from string to integer", key))
	}
	_, err = event.Put(key, itemConverted)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failure updating %v key", key))
	}
	return nil
}

// GetNatsTimestamp gets the timestamp of base level metrics NATS server returns
func GetNatsTimestamp(event mapstr.M) (time.Time, error) {
	var timeStamp time.Time
	timestamp, _ := event.GetValue("server.time")
	timestampString := timestamp.(string)
	timeStamp, err := time.Parse(time.RFC3339, timestampString)
	if err != nil {
		return timeStamp, err
	}
	return timeStamp, nil
}
