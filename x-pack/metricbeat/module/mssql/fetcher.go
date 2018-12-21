// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mssql

import (
	"database/sql"

	"github.com/pkg/errors"

	s "github.com/elastic/beats/libbeat/common/schema"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"

	// Register driver.
	_ "github.com/denisenkom/go-mssqldb"
)

// NewFetcher is called from every metricset to initialize their fetching routines. It opens a different connection to
// the database for each fetch operation
func NewFetcher(uri string, qs []string, schema *s.Schema, log *logp.Logger) (*Fetcher, error) {
	db, err := sql.Open("sqlserver", uri)
	if err != nil {
		return nil, errors.Wrap(err, "could not create db instance")
	}

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

	return f, nil
}

// Fetcher will make queries sequentially to the database to fetch results. It
// must be created by each metricset and fed with the queries that must be
// executed, a SQL implementor and a pointer to the schema to "apply". A
// metricset could, potentially, need more than one query to fill all its
// results.
type Fetcher struct {
	schema *s.Schema // Schema is the metricset schema to apply to the fetched data to turn it into a result mapstr.
	db     *sql.DB   // Database to execute queries against.
	log    *logp.Logger

	queries []string // List of queries to execute concurrently.
}

// Close closes db connection to the server
func (f *Fetcher) Close() error {
	return f.db.Close()
}

// Report receives a mb.ReporterV2 to send the data to the outputs. Because the operations are common between metricsets
// it can be extracted to a common function
func (f *Fetcher) Report(reporter mb.ReporterV2) {
	var err error
	for _, q := range f.queries {
		if err = f.getEventsWithQuery(f.db, q, reporter); err != nil {
			logp.Error(err)
		}
	}
}

type rowsResultHandler struct {
	dest        []interface{}
	rows        *sql.Rows
	reporter    mb.ReporterV2
	columnNames []string
}

func newRowsResult(reporter mb.ReporterV2, columnNames []string, rows *sql.Rows) *rowsResultHandler {
	return &rowsResultHandler{
		dest:        make([]interface{}, len(columnNames)),
		columnNames: columnNames,
		rows:        rows,
		reporter:    reporter,
	}
}

// getEventsWithQuery performs the query on the database and creates a rowsResultHandler to handle them
func (f *Fetcher) getEventsWithQuery(db *sql.DB, query string, reporter mb.ReporterV2) (err error) {
	var rows *sql.Rows
	rows, err = db.Query(query)
	if err != nil {
		return errors.Wrapf(err, "error performing db query='%v'", query)
	}
	defer func() {
		if err2 := rows.Close(); err != nil {
			if err != nil {
				err = errors.Wrap(err, err2.Error())
			} else {
				err = err2
			}
		}
	}()

	columnNames, err := rows.Columns()
	if err != nil {
		return errors.Wrap(err, "error getting column names")
	}

	rowsResult := newRowsResult(reporter, columnNames, rows)

	if err = rowsResult.handle(f.schema); err != nil {
		return errors.Wrap(err, "could not convert rows result to beats mapstr")
	}

	return nil
}

// handle is called with the result of each query.
func (rr *rowsResultHandler) handle(s *s.Schema) (err error) {
	for rr.rows.Next() {
		// We assign pointers to the destination to pass it to Scan.
		rawResult := make([]*string, len(rr.columnNames))
		for i := range rawResult {
			rr.dest[i] = &rawResult[i]
		}

		if err = rr.rows.Scan(rr.dest...); err != nil {
			return errors.Wrap(err, "error scanning row of result")
		}

		// Now we need to get the values of the pointers back into a normal
		// map[string]interface{} to use it with the schema.
		mapOfResults := make(map[string]interface{})

		for i, res := range rawResult {
			if res != nil {
				mapOfResults[rr.columnNames[i]] = *res
			}
		}

		result, err := s.Apply(mapOfResults)
		if err != nil {
			err = errors.Wrap(err, "error trying to apply schema")
			logp.Error(err)
			rr.reporter.Error(err)
			continue
		}

		rr.reporter.Event(mb.Event{MetricSetFields: result})
	}

	return nil
}
