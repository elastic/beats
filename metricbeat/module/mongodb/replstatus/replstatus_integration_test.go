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
// +build integration

package replstatus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
	"github.com/elastic/beats/v7/metricbeat/module/mongodb"
)

func TestFetch(t *testing.T) {
	t.Skip("Flaky test: https://github.com/elastic/beats/issues/29208")
	service := compose.EnsureUp(t, "mongodb")

	err := initiateReplicaSet(t, service.Host())
	if !assert.NoError(t, err) {
		t.FailNow()
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
	oplog := event["oplog"].(common.MapStr)
	allocated := oplog["size"].(common.MapStr)["allocated"].(int64)
	assert.True(t, allocated >= 0)

	used := oplog["size"].(common.MapStr)["used"].(float64)
	assert.True(t, used > 0)

	firstTs := oplog["first"].(common.MapStr)["timestamp"].(int64)
	assert.True(t, firstTs >= 0)

	window := oplog["window"].(int64)
	assert.True(t, window >= 0)

	members := event["members"].(common.MapStr)
	primary := members["primary"].(common.MapStr)
	assert.NotEmpty(t, primary["host"].(string))
	assert.True(t, primary["optime"].(int64) > 0)

	set := event["set_name"].(string)
	assert.Equal(t, set, "beats")
}

func TestData(t *testing.T) {
	service := compose.EnsureUp(t, "mongodb")

	f := mbtest.NewReportingMetricSetV2Error(t, getConfig(service.Host()))
	if err := mbtest.WriteEventsReporterV2Error(f, t, ""); err != nil {
		t.Fatal("write", err)
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
	url := host

	dialInfo, err := mgo.ParseURL(url)
	if err != nil {
		return err
	}
	dialInfo.Direct = true

	mongoSession, err := mongodb.NewDirectSession(dialInfo)
	if err != nil {
		return err
	}
	defer mongoSession.Close()

	// get oplog.rs collection
	db := mongoSession.DB("admin")
	config := ReplicaConfig{"beats", []Host{{0, url}}}
	var initiateResult map[string]interface{}
	if err := db.Run(bson.M{"replSetInitiate": config}, &initiateResult); err != nil {
		if err.Error() != "already initialized" {
			return err
		}
	}

	var status map[string]interface{}
	for {
		db.Run(bson.M{"replSetGetStatus": 1}, &status)
		myState, ok := status["myState"].(int)
		t.Logf("Mongodb state is %d", myState)
		if ok && myState == 1 {
			time.Sleep(5 * time.Second) // hack, wait more for replica set to become stable
			break
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
