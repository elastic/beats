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

/*
Package server fetches metrics from ZooKeeper by using the srvr command

See the srvr command documentation at
https://zookeeper.apache.org/doc/current/zookeeperAdmin.html

ZooKeeper srvr Command Output

  $ echo srvr | nc localhost 2181
	Zookeeper version: 3.4.13-2d71af4dbe22557fda74f9a9b4309b15a7487f03, built on 06/29/2018 04:05 GMT
Latency min/avg/max: 1/2/3
Received: 46
Sent: 45
Connections: 1
Outstanding: 0
Zxid: 0x700601132
Mode: standalone
Node count: 4
Proposal sizes last/min/max: -3/-999/-1


*/
package server

import (
	"bufio"
	"encoding/binary"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/zookeeper"
	"github.com/pkg/errors"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var latencyCapturer = regexp.MustCompile(`(\d+)/(\d+)/(\d+)`)
var ipCapturer = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
var thatNumberCapturer = regexp.MustCompile(`\[(\d+)\]`)
var portCapturer = regexp.MustCompile(`:(\d+)\[`)
var dataCapturer = regexp.MustCompile(`(\w+)=(\d+)`)
var fieldsCapturer = regexp.MustCompile(`^([a-zA-Z\s]+):\s(\d+)`)
var versionCapturer = regexp.MustCompile(`:\s(.*),`)
var dateCapturer = regexp.MustCompile(`built on (.*)`)

func init() {
	mb.Registry.MustAddMetricSet("zookeeper", "server", New,
		mb.WithHostParser(parse.PassThruHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching ZooKeeper health metrics.
type MetricSet struct {
	mb.BaseMetricSet
}

// New creates new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

// Fetch fetches metrics from ZooKeeper by making a tcp connection to the
// command port and sending the "srvr" command and parsing the output.
func (m *MetricSet) Fetch(reporter mb.ReporterV2) {
	outputReader, err := zookeeper.RunCommand("srvr", m.Host(), m.Module().Config().Timeout)
	if err != nil {
		reporter.Error(errors.Wrap(err, "srvr command failed"))
		return
	}

	metricsetFields, err := parseSrvr(outputReader)
	if err != nil {
		reporter.Error(err)
		return
	}

	reporter.Event(mb.Event{
		MetricSetFields: metricsetFields,
	})
}

func parseSrvr(i io.Reader) (common.MapStr, error) {
	scanner := bufio.NewScanner(i)

	output := common.MapStr{}

	//Get version
	ok := scanner.Scan()
	if !ok {
		return nil, errors.New("no initial successful scan, aborting")
	}
	output.Put("version.id", versionCapturer.FindStringSubmatch(scanner.Text())[1])
	output.Put("version.date", dateCapturer.FindStringSubmatch(scanner.Text())[1])

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, "Zxid") {
			xid, err := parseXid(line)
			if err != nil {
				return nil, errors.Wrap(err, "error parsing xid line")
			}
			output.Put("xid", xid)

			continue
		}

		if strings.Contains(line, "Latency") {
			latency, err := parseLatencyLine(line)
			if err != nil {
				return nil, errors.Wrap(err, "error parsing latency values")
			}
			output.Put("latency", latency)

			continue
		}

		if strings.Contains(line, "Proposal sizes") {
			proposalSizes, err := parseProposalSizes(line)
			if err != nil {
				return nil, errors.Wrap(err, "error parsing proposal sizes line")
			}
			output.Put("proposal_sizes", proposalSizes)

			continue
		}

		if strings.Contains(line, "Mode") {
			output.Put("mode", strings.Split(line, " ")[1])
			continue
		}

		// If code reaches here easy to parse lines or blank lines like the following:
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
			val, err := strconv.ParseInt(result[2], 10, 64)
			if err != nil {
				return nil, err
			}
			output.Put(strings.ToLower(strings.Replace(result[1], " ", "_", -1)), val)
		}
	}
	return output, nil
}

func parseXid(line string) (common.MapStr, error){
	output := common.MapStr{}

	zxidString := strings.Split(line, " ")[1]
	zxid, err := strconv.ParseInt(zxidString[2:], 16, 64)
	if err != nil {
		return nil, err
	}

	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, uint64(zxid))

	epoch := bs[:4]
	count := bs[4:]

	output.Put("zxid.original", zxidString)
	output.Put("zxid.epoch", binary.BigEndian.Uint32(epoch))
	output.Put("zxid.count", binary.BigEndian.Uint32(count))

	return output, nil
}

func parseProposalSizes(line string) (common.MapStr, error) {
	output := common.MapStr{}

	values := strings.Split(strings.Split(line, " ")[3], "/")
	last, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return nil, err
	}
	output.Put("last", last)

	min, err := strconv.ParseInt(values[1], 10, 64)
	if err != nil {
		return nil, err
	}
	output.Put("min", min)

	max, err := strconv.ParseInt(values[2], 10, 64)
	if err != nil {
		return nil, err
	}
	output.Put("max", max)

	return output, nil
}

func parseLatencyLine(line string) (common.MapStr, error) {
	output := common.MapStr{}

	values := latencyCapturer.FindStringSubmatch(line)

	min, err := strconv.ParseInt(values[1], 10, 64)
	if err != nil {
		return nil, err
	}
	output.Put("min", min)

	avg, err := strconv.ParseInt(values[2], 10, 64)
	if err != nil {
		return nil, err
	}
	output.Put("avg", avg)

	max, err := strconv.ParseInt(values[3], 10, 64)
	if err != nil {
		return nil, err
	}
	output.Put("max", max)

	return output, nil
}
