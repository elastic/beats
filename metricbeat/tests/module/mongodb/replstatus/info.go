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

package replstatus

import (
	"errors"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type oplogInfo struct {
	allocated int64
	used      float64
	firstTs   int64
	lastTs    int64
	diff      int64
}

// CollSize contains data about collection size
type CollSize struct {
	MaxSize int64   `bson:"maxSize"` // Shows the maximum size of the collection.
	Size    float64 `bson:"size"`    // The total size in memory of all records in a collection.
}

const oplogCol = "oplog.rs"

func getReplicationInfo(mongoSession *mgo.Session) (*oplogInfo, error) {
	// get oplog.rs collection
	db := mongoSession.DB("local")
	if collections, err := db.CollectionNames(); err != nil || !contains(collections, oplogCol) {
		if err == nil {
			err = errors.New("collection oplog.rs was not found")
		}

		return nil, err
	}
	collection := db.C(oplogCol)

	// get oplog size
	var oplogSize CollSize
	if err := db.Run(bson.D{{Name: "collStats", Value: oplogCol}}, &oplogSize); err != nil {
		return nil, err
	}

	// get first and last items in the oplog
	firstTs, err := getOpTimestamp(collection, "$natural")
	if err != nil {
		return nil, err
	}

	lastTs, err := getOpTimestamp(collection, "-$natural")
	if err != nil {
		return nil, err
	}

	diff := lastTs - firstTs

	return &oplogInfo{
		allocated: oplogSize.MaxSize,
		used:      oplogSize.Size,
		firstTs:   firstTs,
		lastTs:    lastTs,
		diff:      diff,
	}, nil
}

func getOpTimestamp(collection *mgo.Collection, sort string) (int64, error) {
	iter := collection.Find(nil).Sort(sort).Iter()

	var opTime OpTime
	if !iter.Next(&opTime) {
		return 0, errors.New("objects not found in local.oplog.rs -- Is this a new and empty db instance?")
	}

	return opTime.getTimeStamp(), nil
}

func contains(s []string, x string) bool {
	for _, n := range s {
		if x == n {
			return true
		}
	}
	return false
}
