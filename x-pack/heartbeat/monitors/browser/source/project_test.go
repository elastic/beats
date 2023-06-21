// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.
//go:build linux

package source

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestProjectSource(t *testing.T) {
	type testCase struct {
		name    string
		cfg     mapstr.M
		wantErr bool
	}
	testCases := []testCase{
		{
			"decode project content",
			mapstr.M{
				"content": "UEsDBBQACAAIAJ27qVQAAAAAAAAAAAAAAAAiAAAAZXhhbXBsZXMvdG9kb3MvYWR2YW5jZWQuam91cm5leS50c5VRPW/CMBDd+RWnLA0Sigt0KqJqpbZTN+iEGKzkIC6JbfkuiBTx3+uEEAGlgi7Rnf38viIESCLkR/FJ6Eis1VIjpanATBKrWFCpOUU/kcCNzG2GJNgkhoRM1lLHmERfpnAay4ipo3JrHMMWmjPYwcKZHILn33zBqIV3ADIjkxdrJ4y251eZJFNJq3b1Hh1XJx+KeKK+8XATpxiv3o07RidI7Ex5OOocTEQixcz6mF66MRgGXkmxMhqkTiA2VcJ6NQsgpZcZAnueoAfhFqxcYs9/ncwJdl0YP9XeY6OJgb3qFDcMYwhejb5jsAUDyYxBaSi9HmCJlfZJ2vCYNCpc1h2d5m8AB/r99cU+GmS/hpwXc4nmrKh/K917yK57VqZe1lU6zM26WvIiY2WbHunWIiusb3IWVBP0/bP9NGinYTC/qcqWLloY9ybjNAy5VbzYdP1sdz3+8FqJleqsP7/ONPjjp++TPgS3eaks/wBQSwcIVYIEHGwBAADRAwAAUEsDBBQACAAIAJ27qVQAAAAAAAAAAAAAAAAZAAAAZXhhbXBsZXMvdG9kb3MvaGVscGVycy50c5VUTYvbMBC9768YRGAVyKb0uktCu9CeektvpRCtM4nFKpKQxt2kwf+9I9lJ5cRb6MWW5+u9eTOW3nsXCE4QCf0M8OCxImhhG9wexCc0KpKuPsSjpRr5FMXTXeVsJDBObT57v+I8WID0aoczaIKZwmIJpzvIFaUwqrFVDcp7MQPFdSqQlxAA9aY0QUqe7xw5mQo8saflZ3uGUpvNdxVfh1DEliHWmuOyGSan9GrXY4hdSW19Q1yswJ9Ika1zi28P5DZOZCZnjp2Pjh5lhr71+YAxSvHFEgZx20UqGVdoWGAXGFo0Zp5sD0YnOXX+uMi71TY3nTh2PYy0HZCaYMsm0umrC2cYuWYpStwWlksgPNBC9CKJ9UDqGDFQAv7GrFb6N/aqD0hEtl9pX9VYvQLViroR5KZqFXmlVEXmyDNJWS0wkT1aiqPD6fZPynIsEznoYDqdG7Q7qqcs2DPKzOVG7EyHhSj25n0Zyw62PJvcwH2vzz1PN3czSrifwHlaZfUbThuMFNzxPyj1GVeE/rHWRr2guaz1e6wu0foSmhPTL3DwiuqFshVDu/D4aPSPjz/FIK1n9dwQOfu3gk7pL9k4jK+M5lk0LBRy9CB7nn2yD+cStfuFQQ5+riK9kJQ3JV9cbCmuh1n6HF3h5LleimS7GkoynWVL5+KWS6h/AFBLBwgvDHpj+wEAAC8FAABQSwECLQMUAAgACACdu6lUVYIEHGwBAADRAwAAIgAAAAAAAAAAACAApIEAAAAAZXhhbXBsZXMvdG9kb3MvYWR2YW5jZWQuam91cm5leS50c1BLAQItAxQACAAIAJ27qVQvDHpj+wEAAC8FAAAZAAAAAAAAAAAAIACkgbwBAABleGFtcGxlcy90b2Rvcy9oZWxwZXJzLnRzUEsFBgAAAAACAAIAlwAAAP4DAAAAAA==",
			},
			false,
		},
		{
			"bad encoded content",
			mapstr.M{
				"content": "12312edasd",
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			psrc, err := dummyPSource(tc.cfg)
			if tc.wantErr {
				err = psrc.Fetch()
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			fetchAndValidate(t, psrc)
		})
	}
}

func TestFetchCaching(t *testing.T) {
	cfg := mapstr.M{
		"content": "UEsDBBQACAAIAJ27qVQAAAAAAAAAAAAAAAAiAAAAZXhhbXBsZXMvdG9kb3MvYWR2YW5jZWQuam91cm5leS50c5VRPW/CMBDd+RWnLA0Sigt0KqJqpbZTN+iEGKzkIC6JbfkuiBTx3+uEEAGlgi7Rnf38viIESCLkR/FJ6Eis1VIjpanATBKrWFCpOUU/kcCNzG2GJNgkhoRM1lLHmERfpnAay4ipo3JrHMMWmjPYwcKZHILn33zBqIV3ADIjkxdrJ4y251eZJFNJq3b1Hh1XJx+KeKK+8XATpxiv3o07RidI7Ex5OOocTEQixcz6mF66MRgGXkmxMhqkTiA2VcJ6NQsgpZcZAnueoAfhFqxcYs9/ncwJdl0YP9XeY6OJgb3qFDcMYwhejb5jsAUDyYxBaSi9HmCJlfZJ2vCYNCpc1h2d5m8AB/r99cU+GmS/hpwXc4nmrKh/K917yK57VqZe1lU6zM26WvIiY2WbHunWIiusb3IWVBP0/bP9NGinYTC/qcqWLloY9ybjNAy5VbzYdP1sdz3+8FqJleqsP7/ONPjjp++TPgS3eaks/wBQSwcIVYIEHGwBAADRAwAAUEsDBBQACAAIAJ27qVQAAAAAAAAAAAAAAAAZAAAAZXhhbXBsZXMvdG9kb3MvaGVscGVycy50c5VUTYvbMBC9768YRGAVyKb0uktCu9CeektvpRCtM4nFKpKQxt2kwf+9I9lJ5cRb6MWW5+u9eTOW3nsXCE4QCf0M8OCxImhhG9wexCc0KpKuPsSjpRr5FMXTXeVsJDBObT57v+I8WID0aoczaIKZwmIJpzvIFaUwqrFVDcp7MQPFdSqQlxAA9aY0QUqe7xw5mQo8saflZ3uGUpvNdxVfh1DEliHWmuOyGSan9GrXY4hdSW19Q1yswJ9Ika1zi28P5DZOZCZnjp2Pjh5lhr71+YAxSvHFEgZx20UqGVdoWGAXGFo0Zp5sD0YnOXX+uMi71TY3nTh2PYy0HZCaYMsm0umrC2cYuWYpStwWlksgPNBC9CKJ9UDqGDFQAv7GrFb6N/aqD0hEtl9pX9VYvQLViroR5KZqFXmlVEXmyDNJWS0wkT1aiqPD6fZPynIsEznoYDqdG7Q7qqcs2DPKzOVG7EyHhSj25n0Zyw62PJvcwH2vzz1PN3czSrifwHlaZfUbThuMFNzxPyj1GVeE/rHWRr2guaz1e6wu0foSmhPTL3DwiuqFshVDu/D4aPSPjz/FIK1n9dwQOfu3gk7pL9k4jK+M5lk0LBRy9CB7nn2yD+cStfuFQQ5+riK9kJQ3JV9cbCmuh1n6HF3h5LleimS7GkoynWVL5+KWS6h/AFBLBwgvDHpj+wEAAC8FAABQSwECLQMUAAgACACdu6lUVYIEHGwBAADRAwAAIgAAAAAAAAAAACAApIEAAAAAZXhhbXBsZXMvdG9kb3MvYWR2YW5jZWQuam91cm5leS50c1BLAQItAxQACAAIAJ27qVQvDHpj+wEAAC8FAAAZAAAAAAAAAAAAIACkgbwBAABleGFtcGxlcy90b2Rvcy9oZWxwZXJzLnRzUEsFBgAAAAACAAIAlwAAAP4DAAAAAA==",
	}
	psrc, err := dummyPSource(cfg)
	require.NoError(t, err)
	defer psrc.Close()

	err = psrc.Fetch()
	require.NoError(t, err)
	wdir := psrc.Workdir()
	err = psrc.Fetch()
	require.NoError(t, err)
	wdirNext := psrc.Workdir()
	require.Equal(t, wdir, wdirNext)
}

func validateFileContents(t *testing.T, dir string) {
	expected := []string{
		"examples/todos/helpers.ts",
		"examples/todos/advanced.journey.ts",
		"package.json",
	}
	for _, file := range expected {
		stat, err := os.Stat(path.Join(dir, file))
		assert.NoError(t, err)
		// Permissions should be (rwxrwx---), for running when process has changed its UID
		// note that the files themselves should not have the setuid bit set
		mode := stat.Mode().Perm()
		require.Equalf(t, mode, os.FileMode(0770), "file %v has wrong permissions: expected=%v actual=%v",
			stat.Name(), os.FileMode(0770), mode)
	}
}

func fetchAndValidate(t *testing.T, psrc *ProjectSource) {
	defer func() {
		_ = psrc.Close()
	}()
	err := psrc.Fetch()
	require.NoError(t, err)

	dir, err := os.Stat(psrc.Workdir())
	require.NoError(t, err)

	// Permissions should be (rwxrwx---), for running when process has changed its UID
	// note that the files themselves should not have the setuid bit set
	mode := dir.Mode().Perm()
	require.Equalf(t, mode, os.FileMode(0770), "file %v has wrong permissions: expected=%v actual=%v",
		dir.Name(), os.FileMode(0770), mode)

	validateFileContents(t, psrc.Workdir())
	// check if the working directory is deleted
	require.NoError(t, psrc.Close())
	_, err = os.Stat(psrc.TargetDirectory)
	require.True(t, os.IsNotExist(err), "TargetDirectory %s should have been deleted", psrc.TargetDirectory)
}

func dummyPSource(conf map[string]interface{}) (*ProjectSource, error) {
	psrc := &ProjectSource{}
	y, _ := yaml.Marshal(conf)
	c, err := config.NewConfigWithYAML(y, string(y))
	if err != nil {
		return nil, err
	}
	err = c.Unpack(psrc)
	return psrc, err
}
