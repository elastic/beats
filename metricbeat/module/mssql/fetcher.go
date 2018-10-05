// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mssql

import (
	"database/sql"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	"github.com/pkg/errors"
)

func NewFetcher(config *Config, qs []string, schema *s.Schema) (*fetcher, error) {
	db, err := NewDB(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not create db instance")
	}

	qr := &fetcher{
		queries: qs,
		Schema:  schema,
		db:      db,
	}

	qr.WaitGroup.Add(len(qs))

	qr.doQueries()

	return qr, nil
}

// fetcher will make queries concurrently to the database to fetch results. It must be created by each
// metricset and fed with the queries that must be executed, a SQL implementor and a pointer to the schema to "apply".
// A metricset could, potentially, need more than one query to fill all its results.
type fetcher struct {
	//Maprs is the list of metrics returned by the database
	Maprs []common.MapStr

	// Schema is the metricset schema to apply the fetched data into
	Schema *s.Schema

	// Error will accumulate errors returned by each concurrent query in a thread safe manner
	Error error

	queries []string
	db      *sql.DB
	sync.Mutex
	sync.WaitGroup
}

// doQueries is executed on object creation from the metricsets via NewFetcher. It will launch a goroutine for each
// query stored and wait for all results before returning.
func (f *fetcher) doQueries() {
	for _, q := range f.queries {
		go f.doEventsFetcherQuery(q)
	}
	f.Wait()
}

// doEventsFetcherQuery adds common.Mapr objects concurrently
func (f *fetcher) doEventsFetcherQuery(q string) {
	defer f.Done()

	maprSlice, err := f.getEventsWithQuery(f.db, q)
	if err != nil {
		// Wrap the error if any error already exists or set it
		f.Lock()
		defer f.Unlock()

		if f.Error == nil {
			f.Error = err
		} else {
			f.Error = errors.Wrap(f.Error, err.Error())
		}

		return
	}

	f.Lock()
	defer f.Unlock()

	f.Maprs = append(f.Maprs, maprSlice...)
}

//Close implements io.Close to close the database connection.
func (f *fetcher) Close() error {
	return f.db.Close()
}

// getEventsWithQuery performs the query on the database and converts the rows to an slice of common.Mapr
func (f *fetcher) getEventsWithQuery(db *sql.DB, query string) ([]common.MapStr, error) {
	// Returns the global status, also for versions previous 5.0.2
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrap(err, "error doing query to database")
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			//TODO Log error? Ignore it?
		}
	}()

	mapR, err := f.eventsMapping(rows)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert row result to beats mapr")
	}

	return mapR, nil
}

// eventsMapping takes a *sql.Rows to convert them to a []common.Mapr slice dynamically without knowing the column names
// in advance.
func (f *fetcher) eventsMapping(rows *sql.Rows) ([]common.MapStr, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "error getting column names")
	}

	totalColumns := len(columnNames)

	dest := make([]interface{}, totalColumns)
	results := make([]common.MapStr, 0)

	totalRowResults := 0
	for rows.Next() {
		totalRowResults++

		// We assign pointers to the destination to pass it to Scan
		rawResult := make([]*string, totalColumns)
		for i := range rawResult {
			dest[i] = &rawResult[i]
		}

		if err = rows.Scan(dest...); err != nil {
			return nil, errors.Wrap(err, "error scanning first row of result")
		}

		//Now we need to get the values of the pointers back into a normal map[string]interface{} to use it with the schema
		mapOfResults := make(map[string]interface{})

		for i, res := range rawResult {
			if res != nil {
				mapOfResults[columnNames[i]] = *res
			}
		}

		f.Lock()
		res, _ := f.Schema.Apply(mapOfResults)
		results = append(results, res)
		f.Unlock()
	}

	if totalRowResults == 0 {
		return nil, errors.New("no results found")
	}

	return results, nil
}
