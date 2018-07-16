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

package oplog

import (
	"errors"

	"gopkg.in/mgo.v2/bson"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mongodb"
)

const oplogCol = "oplog.rs"

var debugf = logp.MakeDebug("mongodb.oplog")

func init() {
	mb.Registry.MustAddMetricSet("mongodb", "oplog", New,
		mb.WithHostParser(mongodb.ParseURL),
		mb.DefaultMetricSet())
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	*mongodb.MetricSet
}

func contains(s []string, x string) bool {
	for _, n := range s {
		if x == n {
			return true
		}
	}
	return false
}

// New creates a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Experimental("The mongodb oplog metricset is experimental.")

	ms, err := mongodb.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch() (common.MapStr, error) {
	// instantiate direct connections to each of the configured Mongo hosts
	mongoSession, err := mongodb.NewDirectSession(m.DialInfo)
	if err != nil {
		return nil, err
	}
	defer mongoSession.Close()

	// get oplog.rs collection
	db := mongoSession.DB("local")
	if collections, err := db.CollectionNames(); err != nil || !contains(collections, oplogCol) {
		if err == nil {
			err = errors.New("Collection oplog.rs was not found")
		}

		logp.Err(err.Error())
		return nil, err
	}
	collection := db.C(oplogCol)

	//  oplog size
	var oplogStatus map[string]interface{}
	if err := db.Run(bson.D{{Name: "collStats", Value: oplogCol}}, &oplogStatus); err != nil {
		return nil, err
	}

	allocated := oplogStatus["maxSize"].(int64)
	used := int64(oplogStatus["size"].(float64))

	// get first and last items in the oplog
	oplogIter := collection.Find(nil).Sort("$natural").Iter()
	oplogReverseIter := collection.Find(nil).Sort("-$natural").Iter()
	var first, last interface{}
	if !oplogIter.Next(&first) || !oplogReverseIter.Next(&last) {
		err := errors.New("Objects not found in local.oplog.rs -- Is this a new and empty db instance?")
		logp.Err(err.Error())
		return nil, err
	}

	firstTsValue, firstOk := first.(bson.M)["ts"].(bson.MongoTimestamp)
	lastTsValue, lastOk := last.(bson.M)["ts"].(bson.MongoTimestamp)
	if !firstOk || !lastOk {
		err := errors.New("Unexpected timestamp value found in first/last oplog item")
		return nil, err
	}
	firstTs := int64(firstTsValue)
	lastTs := int64(lastTsValue)
	diff := lastTs - firstTs

	result := map[string]interface{}{
		"logSize":  allocated,
		"used":     used,
		"tFirst":   firstTs,
		"tLast":    lastTs,
		"timeDiff": diff,
	}
	event, _ := schema.Apply(result)

	return event, nil
}
