// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package billing

import (
	"io/ioutil"
	"log"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentMonth(t *testing.T) {
	currentMonth := getCurrentMonth()
	_, err := strconv.ParseInt(currentMonth, 0, 64)
	assert.NoError(t, err)
}

func TestGenerateQuery(t *testing.T) {
	log.SetOutput(ioutil.Discard)

	query := generateQuery("my-table", "jan", "cost")
	log.Println(query)

	// verify that table name quoting is in effect
	assert.Contains(t, query, "`my-table`")
	// verify the group by is preserved
	assert.Contains(t, query, "GROUP BY 1, 2, 3, 4, 5")
	// verify the order by is preserved
	assert.Contains(t, query, "ORDER BY 1 ASC, 2 ASC, 3 ASC, 4 ASC, 5 ASC")
}
