// +build !integration

package mysql

import (
	"testing"
	"time"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDSN(t *testing.T) {
	const query = "?readTimeout=10s&timeout=10s&writeTimeout=10s"

	var tests = []struct {
		host     string
		username string
		password string
		uri      string
	}{
		{"", "", "", "tcp(127.0.0.1:3306)/" + query},
		{"", "root", "secret", "root:secret@tcp(127.0.0.1:3306)/" + query},
		{"unix(/tmp/mysql.sock)/", "root", "", "root@unix(/tmp/mysql.sock)/" + query},
		{"tcp(127.0.0.1:3306)/", "", "", "tcp(127.0.0.1:3306)/" + query},
		{"tcp(127.0.0.1:3306)/", "root", "", "root@tcp(127.0.0.1:3306)/" + query},
		{"tcp(127.0.0.1:3306)/", "root", "secret", "root:secret@tcp(127.0.0.1:3306)/" + query},
	}

	for _, test := range tests {
		c := map[string]interface{}{
			"username": test.username,
			"password": test.password,
		}
		mod := mbtest.NewTestModule(t, c)
		mod.ModConfig.Timeout = 10 * time.Second

		hostData, err := ParseDSN(mod, test.host)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.uri, hostData.URI)
		if test.username != "" {
			assert.NotContains(t, hostData.SanitizedURI, test.username)
		}
		if test.password != "" {
			assert.NotContains(t, hostData.SanitizedURI, test.password)
		}
	}
}
