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
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

	collections, err := db.ListCollectionNames(context.Background(), bson.D{})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve collection names: %w", err)
	}

	if !contains(collections, oplogCol) {
		return nil, errors.New("collection oplog.rs was not found")
	}

	collection := db.Collection(oplogCol)

	// get oplog size
	var oplogSize CollSize
	res := db.RunCommand(context.Background(), bson.D{bson.E{Key: "collStats", Value: oplogCol}})
	if err = res.Err(); err != nil {
		return nil, fmt.Errorf("'collStats' command returned an error: %w", err)
	}

	if err = res.Decode(&oplogSize); err != nil {
		return nil, fmt.Errorf("could not decode mongodb op log size: %w", err)
	}

	// get first and last items in the oplog
	firstTs, lastTs, err := getOpTimestamp(collection)
	if err != nil {
		return nil, fmt.Errorf("could not get operation timestamp in op log: %w", err)
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

func getOpTimestamp(collection *mongo.Collection) (uint32, uint32, error) {

	// Find both first and last timestamps using $min and $max
	pipeline := bson.A{
		bson.M{"$group": bson.M{"_id": 1, "minTS": bson.M{"$min": "$ts"}, "maxTS": bson.M{"$max": "$ts"}}},
	}

	cursor, err := collection.Aggregate(context.Background(), pipeline)
	if err != nil {
		return 0, 0, fmt.Errorf("could not get operation timestamps in op log: %w", err)
	}
	defer cursor.Close(context.Background())

	var result struct {
		MinTS time.Time `bson:"minTS"`
		MaxTS time.Time `bson:"maxTS"`
	}

	if !cursor.Next(context.Background()) {
		return 0, 0, errors.New("no documents found in op log")
	}
	if err := cursor.Decode(&result); err != nil {
		return 0, 0, fmt.Errorf("error decoding response for timestamps: %w", err)
	}

	minTS := uint32(result.MinTS.Unix())
	maxTS := uint32(result.MaxTS.Unix())

	return minTS, maxTS, nil
}

func contains(s []string, x string) bool {
	for _, n := range s {
		if x == n {
			return true
		}
	}
	return false
}
