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

//go:build integration

package replstatus

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/mongodb"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestFetch(t *testing.T) {
	t.Skip("Flaky Test: https://github.com/elastic/beats/issues/31768")

	service := compose.EnsureUp(t, "mongodb")

	err := initiateReplicaSet(t, service.Host())
	if err != nil {
		t.Skipf("(skipping test) initialization of mongo replica failed: %s", err.Error())
	}

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	events, errs := mbtest.ReportingFetchV2Error(f)
	if len(errs) > 0 {
		t.Fatalf("Expected 0 error, had %d. %v\n", len(errs), errs)
	}
	assert.NotEmpty(t, events)
	event := events[0].MetricSetFields

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), events[0])

	// Check event fields
	oplog := event["oplog"].(mapstr.M)
	allocated := oplog["size"].(mapstr.M)["allocated"].(int64)
	assert.True(t, allocated >= 0)

	used := oplog["size"].(mapstr.M)["used"].(float64)
	assert.True(t, used > 0)

	firstTs := oplog["first"].(mapstr.M)["timestamp"].(uint32)
	assert.True(t, firstTs >= 0)

	window := oplog["window"].(uint32)
	assert.True(t, window >= 0)

	members := event["members"].(mapstr.M)
	primary := members["primary"].(mapstr.M)
	assert.NotEmpty(t, primary["host"].(string))
	assert.True(t, primary["optime"].(int64) > 0)

	set := event["set_name"].(string)
	assert.Equal(t, set, "beats")
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	err := initiateReplicaSet(t, service.Host())
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("could not generate data.json file:", err)
	}
}

func getConfig(host string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "mongodb",
		"metricsets": []string{"replstatus"},
		"hosts":      []string{host},
	}
}

func initiateReplicaSet(t *testing.T, host string) error {
	uri := "mongodb://" + host
	client, err := mongodb.NewClient(mongodb.ModuleConfig{
		Hosts: []string{host},
	}, uri, time.Second*5, readpref.PrimaryMode)
	if err != nil {
		return fmt.Errorf("could not create mongodb client: %w", err)
	}

	defer func() {
		client.Disconnect(context.Background())
	}()

	// get oplog.rs collection
	db := client.Database("admin")
	config := ReplicaConfig{"beats", []Host{{0, host}}}
	res := db.RunCommand(context.Background(), bson.M{"replSetInitiate": config})
	if err = res.Err(); err != nil {
		// Maybe it is already initialized?
		errorString := strings.ToLower(err.Error())
		if strings.Contains(strings.ToLower(errorString), "already") &&
			strings.Contains(errorString, "initialized") {
			return nil
		}
	}

	return nil
}

type ReplicaConfig struct {
	id      string `bson:_id`
	members []Host `bson:hosts`
}

type Host struct {
	id   int    `bson:_id`
	host string `bson:host`
}
