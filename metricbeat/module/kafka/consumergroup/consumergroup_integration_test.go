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

// +build integration

package consumergroup

import (
	"io"
	"net"
	"testing"
	"time"

	saramacluster "github.com/bsm/sarama-cluster"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/kafka/mtest"
)

func TestData(t *testing.T) {
	mtest.DataRunner.Run(t, compose.Suite{"Data": func(t *testing.T, host string) {
		c, err := startConsumer(t, "metricbeat-test", host)
		if err != nil {
			t.Fatal(errors.Wrap(err, "starting kafka consumer"))
		}
		defer c.Close()

		ms := mbtest.NewReportingMetricSetV2(t, getConfig(host))
		for retries := 0; retries < 3; retries++ {
			err = mbtest.WriteEventsReporterV2(ms, t, "")
			if err == nil {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Fatal("write", err)
	}})
}

func startConsumer(t *testing.T, topic, host string) (io.Closer, error) {
	brokers := []string{net.JoinHostPort(host, "9092")}
	topics := []string{topic}
	return saramacluster.NewConsumer(brokers, "test-group", topics, nil)
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "kafka",
		"metricsets": []string{"consumergroup"},
		"hosts":      []string{net.JoinHostPort(host, "9092")},
	}
}
