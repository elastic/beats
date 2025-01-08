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

func getReplicationInfo(client *mongo.Client) (*oplogInfo, error) {
	// get oplog.rs collection
	db := client.Database("local")

	// Get oplog size using collStats - this is lightweight
	var oplogSize CollSize
	res := db.RunCommand(context.Background(), bson.D{
		{Key: "collStats", Value: oplogCol},
	})
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("collStats command failed: %w", err)
	}
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

func getOpTimestamp(collection *mongo.Collection) (uint32, uint32, error) {
	// Use natural order for efficiency
	var firstDoc struct {
		Timestamp time.Time `bson:"ts"`
	}

	// Get oldest entry using natural order
	firstOpts := options.FindOne().SetProjection(bson.D{{Key: "ts", Value: 1}})
	err := collection.FindOne(context.TODO(), bson.D{}, firstOpts).Decode(&firstDoc)
	if err != nil {
		return 0, 0, fmt.Errorf("first timestamp query failed: %w", err)
	}

	// Get newest entry using reverse natural order
	var lastDoc struct {
		Timestamp time.Time `bson:"ts"`
	}
	lastOpts := options.FindOne().
		SetProjection(bson.D{{Key: "ts", Value: 1}}).
		SetSort(bson.D{{Key: "$natural", Value: -1}})
	err = collection.FindOne(context.TODO(), bson.D{}, lastOpts).Decode(&lastDoc)
	if err != nil {
		return 0, 0, fmt.Errorf("last timestamp query failed: %w", err)
	}

	return uint32(firstDoc.Timestamp.Unix()), uint32(lastDoc.Timestamp.Unix()), nil
}
