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
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type oplogInfo struct {
	allocated int64
	used      float64
	firstTs   uint32
	lastTs    uint32
	diff      uint32
}

// CollSize contains data about collection size
type CollSize struct {
	MaxSize int64   `bson:"maxSize"` // Shows the maximum size of the collection.
	Size    float64 `bson:"size"`    // The total size in memory of all records in a collection.
}

const oplogCol = "oplog.rs"

// getReplicationInfo returns oplog info from local.oplog.rs
func getReplicationInfo(client *mongo.Client) (*oplogInfo, error) {
	// Get oplog collection info from local.oplog.rs (<db>.<collection>)
	db := client.Database("local")

	// NOTE(shmsr):
	// https://www.mongodb.com/docs/manual/reference/command/collStats/#syntax
	// "scale" field is ommitted here as it is by default 1, i.e., it return sizes in bytes.
	//
	// Also, note that collStats is deprecated since v6.2 but as we support older
	// versions i.e., >= 5.0, let's keep it for now as this still works.
	// TODO(shmsr): For newers versions, we can use db.collection.stats()
	// https://www.mongodb.com/docs/manual/reference/method/db.collection.stats/#mongodb-method-db.collection.stats
	// or use this: https://github.com/percona/mongodb_exporter/blob/95d1865e34940d0d610bb1fbff9745bc66ddbc73/exporter/collstats_collector.go#L100
	res := db.RunCommand(context.Background(), bson.D{
		{Key: "collStats", Value: oplogCol},
	})
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("collStats command failed: %w", err)
	}

	// Get MaxSize and Size from collStats by using db.runCommand
	var oplogSize CollSize
	if err := res.Decode(&oplogSize); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	collection := db.Collection(oplogCol)
	firstTs, lastTs, err := getOpTimestamp(collection)
	if err != nil {
		return nil, err
	}

	info := &oplogInfo{
		allocated: oplogSize.MaxSize,
		used:      oplogSize.Size,
		firstTs:   firstTs,
		lastTs:    lastTs,
		diff:      lastTs - firstTs,
	}

	return info, nil
}

// getOpTimestamp returns the first and last timestamp of the oplog collection.
func getOpTimestamp(collection *mongo.Collection) (uint32, uint32, error) {
	// NOTE(shmsr):
	//
	// When you do db.getReplicationInfo() in monogo shell (mongosh), you can see
	// 	{
	//		...
	// 		tFirst: 'Tue Jan 07 2025 22:33:28 GMT+0530 (India Standard Time)',
	// 		tLast: 'Wed Jan 08 2025 11:45:07 GMT+0530 (India Standard Time)',
	// 		now: 'Wed Jan 08 2025 11:45:14 GMT+0530 (India Standard Time)'
	// 	}
	// i.e., we get tFirst and tLast from oplog.rs which is the first and last
	// timestamp of the oplog.
	// Source from the same is here:
	// 	https://github.com/mongodb/mongo/blob/20cbee37a0ee4d40b35d08b6a34ade81252f86a8/src/mongo/shell/db.js#L863
	// This is how they calculate tFirst and tLast:
	// 	https://github.com/mongodb/mongo/blob/20cbee37a0ee4d40b35d08b6a34ade81252f86a8/src/mongo/shell/db.js#L889
	// So ideally, we will replicate the same logic here:
	// 	var firstc = ol.find().sort({$natural: 1}).limit(1);
	// 	var lastc = ol.find().sort({$natural: -1}).limit(1);
	//
	// oplog.rs is designed to scanned in natural ($natural) order. So, when
	// querying without any sort, it will return the first entry in natural order.
	// When we sort in reverse natural order (i.e., $natural: -1), it will return
	// the last entry in natural order.
	//
	// The queries use optimizations like FindOne() and SetProjection() to limit
	// the amount of data that needs to be scanned and processed.
	// Projection is used to limit the fields returned in the query results i.e., we
	// only need the timestamp (ts) field. FindOne() is used to retrieve a single
	// document from the collection (limit: 1).

	ctx := context.TODO()

	var firstDoc struct {
		Timestamp time.Time `bson:"ts"`
	}

	// Get oldest entry using natural order
	firstOpts := options.
		FindOne().
		SetProjection(bson.D{{Key: "ts", Value: 1}})
	err := collection.FindOne(ctx, bson.D{}, firstOpts).Decode(&firstDoc)
	if err != nil {
		return 0, 0, fmt.Errorf("first timestamp query failed: %w", err)
	}

	// Get newest entry using reverse natural order
	var lastDoc struct {
		Timestamp time.Time `bson:"ts"`
	}
	lastOpts := options.
		FindOne().
		SetProjection(bson.D{{Key: "ts", Value: 1}}).
		SetSort(bson.D{{Key: "$natural", Value: -1}})
	err = collection.FindOne(ctx, bson.D{}, lastOpts).Decode(&lastDoc)
	if err != nil {
		return 0, 0, fmt.Errorf("last timestamp query failed: %w", err)
	}

	return uint32(firstDoc.Timestamp.Unix()), uint32(lastDoc.Timestamp.Unix()), nil
}
