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

package postgresql

import (
	"context"
	"database/sql"
	"sync"
)

type cacheKey struct {
	driver string
	uri    string
}

type dbCacheType struct {
	dbs  map[cacheKey]*sql.DB
	lock sync.Mutex
}

// DBCache keeps a cache of databases for different drivers and URIs.
var DBCache = dbCacheType{
	dbs: make(map[cacheKey]*sql.DB, 4),
}

func (cache *dbCacheType) getDB(driver, uri string) (db *sql.DB, err error) {
	key := cacheKey{
		driver: driver,
		uri:    uri,
	}
	cache.lock.Lock()
	defer cache.lock.Unlock()

	db, found := cache.dbs[key]
	if found {
		return db, nil
	}

	db, err = sql.Open(driver, uri)
	if db != nil {
		cache.dbs[key] = db
	}
	return db, err
}

// GetConnection opens a connection to a DB identified by driver and URI.
func (cache *dbCacheType) GetConnection(driver, uri string) (conn *sql.Conn, err error) {
	db, err := cache.getDB(driver, uri)
	if err != nil {
		return nil, err
	}
	return db.Conn(context.Background())
}
