package postgresql

import (
	"context"
	"database/sql"
	"sync"
)

type dbEntry struct {
	driver string
	uri    string
}

type dbCacheType struct {
	dbs  map[dbEntry]*sql.DB
	lock sync.Mutex
}

// DBCache keeps a cache of databases for different drivers and URIs.
var DBCache = dbCacheType{
	dbs: make(map[dbEntry]*sql.DB, 4),
}

func (cache *dbCacheType) getDB(driver, uri string) (db *sql.DB, err error) {
	key := dbEntry{
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
