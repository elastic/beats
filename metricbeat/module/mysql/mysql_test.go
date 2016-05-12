// +build !integration

package mysql

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateDSN(t *testing.T) {
	hostname := "tcp(127.0.0.1:3306)/"
	username := "root"
	password := "test"

	dsn, _ := CreateDSN(hostname, username, password, 0)
	assert.Equal(t, "root:test@tcp(127.0.0.1:3306)/", dsn)

	dsn, _ = CreateDSN(hostname, username, "", 0)
	assert.Equal(t, "root@tcp(127.0.0.1:3306)/", dsn)

	dsn, _ = CreateDSN(hostname, "", "", 0)
	assert.Equal(t, "tcp(127.0.0.1:3306)/", dsn)

	dsn, _ = CreateDSN(hostname, "", "", time.Second)
	assert.Equal(t, "tcp(127.0.0.1:3306)/?readTimeout=1s&timeout=1s&writeTimeout=1s", dsn)
}
