// +build !integration

package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateDSN(t *testing.T) {
	hostname := "tcp(127.0.0.1:3306)/"
	username := "root"
	password := "test"

	dsn := CreateDSN(hostname, username, password)
	assert.Equal(t, "root:test@tcp(127.0.0.1:3306)/", dsn)

	dsn = CreateDSN(hostname, username, "")
	assert.Equal(t, "root@tcp(127.0.0.1:3306)/", dsn)

	dsn = CreateDSN(hostname, "", "")
	assert.Equal(t, "tcp(127.0.0.1:3306)/", dsn)
}
