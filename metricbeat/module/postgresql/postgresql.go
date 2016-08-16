/*
Package postgresql is Metricbeat module for PostgreSQL server.
*/
package postgresql

import (
	"database/sql"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/pkg/errors"
)

func QueryStats(db *sql.DB, query string) ([]map[string]interface{}, error) {

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.Wrap(err, "scanning columns")
	}
	vals := make([][]byte, len(columns))
	valPointers := make([]interface{}, len(columns))
	for i, _ := range vals {
		valPointers[i] = &vals[i]
	}

	results := []map[string]interface{}{}

	for rows.Next() {
		err = rows.Scan(valPointers...)
		if err != nil {
			return nil, errors.Wrap(err, "scanning row")
		}

		result := map[string]interface{}{}
		for i, col := range columns {
			result[col] = string(vals[i])
		}

		logp.Debug("postgresql", "Result: %v", result)
		results = append(results, result)
	}
	return results, nil
}
