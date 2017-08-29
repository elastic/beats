// +build integration

package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"
	_ "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestNewDB(t *testing.T) {
	compose.EnsureUp(t, "mysql")

	db, err := NewDB(GetMySQLEnvDSN())
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)
}
