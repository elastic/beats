// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package browserhistory

import (
	"database/sql"
	"strconv"
)

func stringFromNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func formatNullInt64(value sql.NullInt64, formatter func(int64) string) string {
	if !value.Valid {
		return ""
	}
	return formatter(value.Int64)
}

func decimalStringFromNullInt(value sql.NullInt64) string {
	return formatNullInt64(value, func(v int64) string {
		return strconv.FormatInt(v, 10)
	})
}

func boolStringFromNullInt(value sql.NullInt64) string {
	if !value.Valid {
		return ""
	}
	if value.Int64 != 0 {
		return "1"
	}
	return "0"
}
