// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mssql

import (
	"database/sql"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/elastic/beats/libbeat/common"
	s "github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/libbeat/logp"

	// Register driver.
	_ "github.com/denisenkom/go-mssqldb"
)

func NewFetcher(uri string, qs []string, schema *s.Schema, log *logp.Logger) (*Fetcher, error) {
	db, err := sql.Open("sqlserver", uri)
	if err != nil {
		return nil, errors.Wrap(err, "could not create db instance")
	}
	defer db.Close()

	// Check the connection before executing all queries to reduce the number
	// of connection errors that we might encounter.
	if err = db.Ping(); err != nil {
		return nil, err
	}

	f := &Fetcher{
		queries: qs,
		schema:  schema,
		db:      db,
		log:     log,
	}

	// Run queries concurrently.
	f.doQueries()

	if len(f.errs) > 0 {
		f.Error = multierr.Combine(f.errs...)
	}
	return f, nil
}

// Fetcher will make queries concurrently to the database to fetch results. It
// must be created by each metricset and fed with the queries that must be
// executed, a SQL implementor and a pointer to the schema to "apply". A
// metricset could, potentially, need more than one query to fill all its
// results.
type Fetcher struct {
	Results     []common.MapStr // Results is the list of metrics returned by the database
	Error       error           // Error describes all errors that occurred.
	errs        []error         // errs will accumulate errors returned by each concurrent query in a thread safe manner
	resultsLock sync.Mutex      // resultsLock protects results and errs for concurrent access while running the queries.

	schema *s.Schema // Schema is the metricset schema to apply to the fetched data to turn it into a result mapstr.
	db     *sql.DB   // Database to execute queries against.
	log    *logp.Logger

	queries []string       // List of queries to execute concurrently.
	wg      sync.WaitGroup // WaitGroup to wait for all queries to complete.
}

// doQueries is executed on object creation from the metricsets via NewFetcher.
// It will launch a goroutine for each query stored and wait for all results
// before returning.
func (f *Fetcher) doQueries() {
	f.wg.Add(len(f.queries))
	for _, q := range f.queries {
		go func() {
			defer f.wg.Done()
			f.doEventsFetcherQuery(q)
		}()
	}
	f.wg.Wait()
}

// doEventsFetcherQuery adds common.Mapr objects concurrently.
func (f *Fetcher) doEventsFetcherQuery(q string) {
	results, err := f.getEventsWithQuery(f.db, q)
	if err != nil {
		f.resultsLock.Lock()
		defer f.resultsLock.Unlock()

		f.errs = append(f.errs, err)
		return
	}

	f.resultsLock.Lock()
	defer f.resultsLock.Unlock()
	f.Results = append(f.Results, results...)
}

// getEventsWithQuery performs the query on the database and converts the rows
// to an slice of common.MapStr.
func (f *Fetcher) getEventsWithQuery(db *sql.DB, query string) ([]common.MapStr, error) {
	// Returns the global status, also for versions previous 5.0.2
	rows, err := db.Query(query)
	if err != nil {
		return nil, errors.Wrapf(err, "error performing db query='%v'", query)
	}
	defer rows.Close()

	results, err := f.eventsMapping(rows)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert row result to beats mapstr")
	}

	return results, nil
}

// eventsMapping takes a *sql.Rows to convert them to a []common.MapStr slice
// dynamically without knowing the column names in advance.
func (f *Fetcher) eventsMapping(rows *sql.Rows) ([]common.MapStr, error) {
	columnNames, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "error getting column names")
	}

	var results []common.MapStr
	dest := make([]interface{}, len(columnNames))

	for rows.Next() {
		// We assign pointers to the destination to pass it to Scan.
		rawResult := make([]*string, len(columnNames))
		for i := range rawResult {
			dest[i] = &rawResult[i]
		}

		if err = rows.Scan(dest...); err != nil {
			return nil, errors.Wrap(err, "error scanning row of result")
		}

		// Now we need to get the values of the pointers back into a normal
		// map[string]interface{} to use it with the schema.
		mapOfResults := make(map[string]interface{})

		for i, res := range rawResult {
			if res != nil {
				mapOfResults[columnNames[i]] = *res
			}
		}

		result, err := f.schema.Apply(mapOfResults)
		if err != nil {
			// TODO: Handle schema error.
			f.log.Debugw("error applying schema to query result", "error", err)
		}

		f.resultsLock.Lock()
		results = append(results, result)
		f.resultsLock.Unlock()
	}

	return results, nil
}
