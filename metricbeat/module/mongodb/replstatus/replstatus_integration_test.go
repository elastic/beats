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

package replstatus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/tests/compose"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	"github.com/elastic/beats/metricbeat/module/mongodb"
	"github.com/elastic/beats/metricbeat/module/mongodb/mtest"
)

func TestReplStatus(t *testing.T) {
	mtest.Runner.Run(t, compose.Suite{
		"Fetch": testFetch,
		"Data":  testData,
	})
}

func testFetch(t *testing.T, r compose.R) {
	err := initiateReplicaSet(t, r.Host())
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	f := mbtest.NewEventFetcher(t, mtest.GetConfig("replstatus", r.Host()))
	event, err := f.Fetch()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(), event)

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

func testData(t *testing.T, r compose.R) {
	f := mbtest.NewEventFetcher(t, mtest.GetConfig("replstatus", r.Host()))
	err := mbtest.WriteEvent(f, t)
	if err != nil {
		t.Fatal("write", err)
	}
}
func initiateReplicaSet(t *testing.T, url string) error {
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
