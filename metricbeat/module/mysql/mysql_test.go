package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping in short mode, because it requires MySQL")
	}

	db, err := Connect(GetMySQLEnvDSN())
	assert.NoError(t, err)

	err = db.Ping()
	assert.NoError(t, err)
}
