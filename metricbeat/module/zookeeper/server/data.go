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

package server

import (
	"bufio"
	"encoding/binary"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var latencyCapturer = regexp.MustCompile(`(\d+)/(\d+)/(\d+)`)
var ipCapturer = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
var thatNumberCapturer = regexp.MustCompile(`\[(\d+)\]`)
var portCapturer = regexp.MustCompile(`:(\d+)\[`)
var dataCapturer = regexp.MustCompile(`(\w+)=(\d+)`)
var fieldsCapturer = regexp.MustCompile(`^([a-zA-Z\s]+):\s(\d+)`)
var versionCapturer = regexp.MustCompile(`:\s(.*),`)
var dateCapturer = regexp.MustCompile(`built on (.*)`)

func parseSrvr(i io.Reader, logger *logp.Logger) (mapstr.M, string, error) {
	scanner := bufio.NewScanner(i)

	//Get version
	ok := scanner.Scan()

	if !ok {
		return nil, "", errors.New("no initial successful text scan, aborting")
	}

	output := mapstr.M{}

	version := versionCapturer.FindStringSubmatch(scanner.Text())[1]
	dateString := dateCapturer.FindStringSubmatch(scanner.Text())[1]

	date, err := time.Parse("01/02/2006 03:04 GMT", dateString)
	if err != nil {
		logger.Debugf("error trying to parse date '%s'", dateString)
	} else {
		output.Put("version_date", date.Format(time.RFC3339))
	}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "Zxid") {
			xid, err := parseZxid(line)
			if err != nil {
				err = errors.Wrapf(err, "error parsing 'zxid' line '%s'", line)
				logger.Debug(err.Error())
				continue
			}

			output.Update(xid)

			continue
		}

		if strings.Contains(line, "Latency") {
			latency, err := parseLatencyLine(line)
			if err != nil {
				err = errors.Wrapf(err, "error parsing 'latency values' line '%s'", line)
				logger.Debug(err.Error())
				continue
			}

			output.Put("latency", latency)

			continue
		}

		if strings.Contains(line, "Proposal sizes") {
			proposalSizes, err := parseProposalSizes(line)
			if err != nil {
				err = errors.Wrapf(err, "error parsing 'proposal sizes' line '%s'", line)
				logger.Debug(err.Error())
				continue
			}

			output.Put("proposal_sizes", proposalSizes)

			continue
		}

		if strings.Contains(line, "Mode") {
			modeSplit := strings.Split(line, " ")
			if len(modeSplit) < 1 {
				logger.Debugf("no tokens after splitting line '%s'", line)
				continue
			}

			output.Put("mode", modeSplit[1])
			continue
		}

		// If code reaches here, just easy to parse lines or blank lines like the following are left:
		// Received: 46
		//
		// Sent: 45
		// Connections: 1
		// Outstanding: 0
		results := fieldsCapturer.FindAllStringSubmatch(line, -1)
		if len(results) == 0 {
			//probably a blank line
			continue
		}

		for _, result := range results {
			// When submatching, the method returns the original value and the captured values, as you can see in the
			// regexp of fieldsCapturer, they are 2 (so no less than 3, counting original value)
			if len(result) < 3 {
				logger.Debug("less than 3 tokens (%v) when regexp submatching '%s'", result, line)
				continue
			}

			val, err := strconv.ParseInt(result[2], 10, 64)
			if err != nil {
				err = errors.Wrapf(err, "error trying to parse value '%s' as int", result[2])
				logger.Debug(err.Error())
				continue
			}

			output.Put(strings.ToLower(strings.Replace(result[1], " ", "_", -1)), val)
		}
	}

	return output, version, nil
}

func parseZxid(line string) (mapstr.M, error) {
	output := mapstr.M{}

	zxidSplit := strings.Split(line, " ")
	if len(zxidSplit) < 2 {
		return nil, errors.Errorf("less than 2 tokens (%v) after splitting", zxidSplit)
	}

	zxidString := zxidSplit[1]
	if len(zxidString) < 3 {
		return nil, errors.Errorf("less than 3 characters on '%s'", zxidString)
	}
	zxid, err := strconv.ParseInt(zxidString[2:], 16, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to parse value '%s' to int", zxidString[2:])
	}

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(zxid))

	epoch := bs[:4]
	count := bs[4:]

	output.Put("zxid", zxidString)
	output.Put("epoch", binary.BigEndian.Uint32(epoch))
	output.Put("count", binary.BigEndian.Uint32(count))

	return output, nil
}

func parseProposalSizes(line string) (mapstr.M, error) {
	output := mapstr.M{}

	initialSplit := strings.Split(line, " ")
	if len(initialSplit) < 4 {
		return nil, errors.Errorf("less than 4 tokens (%v) after splitting", initialSplit)
	}

	values := strings.Split(initialSplit[3], "/")
	if len(values) < 3 {
		return nil, errors.Errorf("less than 3 tokens (%v) after splitting", values)
	}
	last, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to parse 'last' value as int from '%s'", values[0])
	}
	output.Put("last", last)

	min, err := strconv.ParseInt(values[1], 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to parse 'min' value as int from '%s'", values[1])
	}
	output.Put("min", min)

	max, err := strconv.ParseInt(values[2], 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to parse 'max' value as int from '%s'", values[2])
	}
	output.Put("max", max)

	return output, nil
}

func parseLatencyLine(line string) (mapstr.M, error) {
	output := mapstr.M{}

	values := latencyCapturer.FindStringSubmatch(line)
	if len(values) < 4 {
		return nil, errors.Errorf("less than 4 fields (%v) after splitting", values)
	}

	min, err := strconv.ParseInt(values[1], 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to parse 'min' value '%s' as int", values[1])
	}
	output.Put("min", min)

	avg, err := strconv.ParseInt(values[2], 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to parse 'avg' value '%s' as int", values[2])
	}
	output.Put("avg", avg)

	max, err := strconv.ParseInt(values[3], 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "error trying to parse 'max' value '%s' as int", values[3])
	}
	output.Put("max", max)

	return output, nil
}
