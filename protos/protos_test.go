package protos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProtocolNames(t *testing.T) {
	assert.Equal(t, "unknown", UnknownProtocol.String())
	assert.Equal(t, "http", HttpProtocol.String())
	assert.Equal(t, "mysql", MysqlProtocol.String())
	assert.Equal(t, "redis", RedisProtocol.String())
	assert.Equal(t, "pgsql", PgsqlProtocol.String())
	assert.Equal(t, "thrift", ThriftProtocol.String())
	assert.Equal(t, "mongodb", MongodbProtocol.String())

	assert.Equal(t, "impossible", Protocol(100).String())
}
