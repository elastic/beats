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
//
// This file was contributed to by generative AI

package bbolt

import (
	"encoding/json"
	"time"

	"go.etcd.io/bbolt"
)

type metadata struct {
	LastAccess int64 `json:"last_access"`
	LastChange int64 `json:"last_change"`
}

func (s *store) updateAccessTime(tx *bbolt.Tx, key string) error {
	bucket := tx.Bucket(bucketMetadata)
	if bucket == nil {
		return nil
	}

	now := time.Now().UnixNano()

	var meta metadata
	if v := bucket.Get([]byte(key)); v != nil {
		if err := json.Unmarshal(v, &meta); err != nil {
			// Best-effort: keep scanning; rewrite metadata from scratch.
			meta = metadata{}
		}
	}
	meta.LastAccess = now

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), data)
}

func (s *store) updateMetadata(tx *bbolt.Tx, key string, changeTime bool) error {
	bucket := tx.Bucket(bucketMetadata)
	if bucket == nil {
		return nil
	}

	now := time.Now().UnixNano()

	meta := metadata{
		LastAccess: now,
	}

	if changeTime {
		meta.LastChange = now
	} else if v := bucket.Get([]byte(key)); v != nil {
		var existing metadata
		if err := json.Unmarshal(v, &existing); err == nil {
			meta.LastChange = existing.LastChange
		}
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(key), data)
}
