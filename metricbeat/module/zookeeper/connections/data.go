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

package connections

import (
	"bufio"
	"io"
	"regexp"
	"strconv"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
)

var ipCapturer = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
var thatNumberCapture = regexp.MustCompile(`\[(\d+)\]`)
var portCapture = regexp.MustCompile(`:(\d+)\[`)
var queueCapture = regexp.MustCompile(`queued=(\d*),`)
var receivedCapture = regexp.MustCompile(`recved=(\d*),`)
var sentCapture = regexp.MustCompile(`sent=(\d*)`)

func init() {
	mb.Registry.MustAddMetricSet("zookeeper", "server", New,
		mb.WithHostParser(parse.PassThruHostParser),
		mb.DefaultMetricSet(),
	)
}

func parseCons(i io.Reader) (common.MapStr, error) {
	scanner := bufio.NewScanner(i)

	output := common.MapStr{}

	for scanner.Scan() {
		line := scanner.Text()

		err := checkRegexSliceAndSetString(output, ipCapturer.FindStringSubmatch(line), "ip", 0)
		if err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse ip"))
		}
		if err = checkRegexSliceAndSetInt(output, portCapture.FindStringSubmatch(line), "port", 1); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse port"))
		}
		if err = checkRegexSliceAndSetInt(output, thatNumberCapture.FindStringSubmatch(line), "number", 1); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse 'number' field"))
		}
		if err = checkRegexSliceAndSetInt(output, queueCapture.FindStringSubmatch(line), "queued", 1); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse 'queued' field"))
		}
		if err = checkRegexSliceAndSetInt(output, receivedCapture.FindStringSubmatch(line), "received", 1); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse 'received' field"))
		}
		if err = checkRegexSliceAndSetInt(output, sentCapture.FindStringSubmatch(line), "sent", 1); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse 'send' field"))
		}
	}

	return output, nil
}

func checkRegexSliceAndSetInt(output common.MapStr, slice []string, key string, index int) error {
	if len(slice) != 0 {
		n, err := strconv.ParseInt(slice[index], 10, 64)
		if err != nil {
			return err
		}
		output.Put(key, n)
	} else {
		return errors.Errorf("%s not found in '%#v'", key, slice)
	}

	return nil
}

func checkRegexSliceAndSetString(output common.MapStr, slice []string, key string, index int) error {
	if len(slice) != 0 {
		output.Put(key, slice[index])
	} else {
		return errors.Errorf("%s not found in '%#v'", key, slice)
	}

	return nil
}
