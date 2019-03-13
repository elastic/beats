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
)

var ipCapturer = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
var portCapture = regexp.MustCompile(`:(\d+)\[`)
var queueCapture = regexp.MustCompile(`queued=(\d*),`)
var receivedCapture = regexp.MustCompile(`recved=(\d*),`)
var sentCapture = regexp.MustCompile(`sent=(\d*)`)

func parseCons(i io.Reader) ([]common.MapStr, error) {
	scanner := bufio.NewScanner(i)

	result := make([]common.MapStr, 0)

	for scanner.Scan() {
		outputLine := common.MapStr{}
		line := scanner.Text()

		// Track parsing information to not send an completely empty line at the end of the for-loop
		oneParsingIsCorrect := false
		err := checkRegexSliceAndSetString(outputLine, ipCapturer.FindStringSubmatch(line), "ip", 0, &oneParsingIsCorrect)
		if err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse ip"))
		}
		if err = checkRegexSliceAndSetInt(outputLine, portCapture.FindStringSubmatch(line), "port", 1, &oneParsingIsCorrect); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse port"))
		}
		if err = checkRegexSliceAndSetInt(outputLine, queueCapture.FindStringSubmatch(line), "queued", 1, &oneParsingIsCorrect); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse 'queued' field"))
		}
		if err = checkRegexSliceAndSetInt(outputLine, receivedCapture.FindStringSubmatch(line), "received", 1, &oneParsingIsCorrect); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse 'received' field"))
		}
		if err = checkRegexSliceAndSetInt(outputLine, sentCapture.FindStringSubmatch(line), "sent", 1, &oneParsingIsCorrect); err != nil {
			logger.Error(errors.Wrap(err, "error trying to parse 'send' field"))
		}

		if oneParsingIsCorrect {
			result = append(result, outputLine)
		}
	}

	return result, nil
}

func checkRegexSliceAndSetInt(output common.MapStr, slice []string, key string, index int, correct *bool) error {
	if len(slice) != 0 {
		n, err := strconv.ParseInt(slice[index], 10, 64)
		if err != nil {
			return err
		}
		*correct = true
		if _, err = output.Put(key, n); err != nil {
			return errors.Wrapf(err, "error placing key '%s' on event", key)
		}
	} else {
		return errors.Errorf("%s not found in '%#v'", key, slice)
	}

	return nil
}

func checkRegexSliceAndSetString(output common.MapStr, slice []string, key string, index int, correct *bool) error {
	if len(slice) != 0 {
		*correct = true
		if _, err := output.Put(key, slice[index]); err != nil {
			return errors.Wrapf(err, "error placing key '%s' on event", key)
		}
	} else {
		return errors.Errorf("%s not found in '%#v'", key, slice)
	}

	return nil
}
