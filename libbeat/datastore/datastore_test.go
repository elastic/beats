// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package datastore

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenClose(t *testing.T) {
	ds := New(dbPath(t), 0640).(*boltDatastore)
	assert.Nil(t, ds.db, "db is created when a bucket is open")

	bucket, err := ds.OpenBucket("test")
	require.NoError(t, err)
	assert.NotNil(t, ds.db, "db should have been created at this point")

	bucket.Close()
	assert.Nil(t, ds.db, "db should be released after all buckets are closed")

	bucket, err = ds.OpenBucket("test")
	require.NoError(t, err, "db should work after being closed")
	bucket.Close()
}

func TestOpenFailureReleaseDB(t *testing.T) {
	ds := New(dbPath(t), 0640).(*boltDatastore)
	assert.Nil(t, ds.db, "db is created when a bucket is open")

	_, err := ds.OpenBucket("")
	require.Error(t, err, "open bucket is expected to require a bucket name")

	assert.Nil(t, ds.db, "there should not be a db connection after an open bucket failure")
}

func TestDataPersistence(t *testing.T) {
	ds := New(dbPath(t), 0640)

	bucket, err := ds.OpenBucket("test")
	require.NoError(t, err)

	bucket2, err := ds.OpenBucket("test")
	require.NoError(t, err)

	somekey := "somekey"
	something := []byte("something")

	bucket.Store(somekey, something)

	err = bucket2.Load(somekey, func(value []byte) error {
		assert.Equal(t, something, value)
		return nil
	})
	assert.NoError(t, err)
	bucket.Close()
	bucket2.Close()

	bucket, err = ds.OpenBucket("test")
	require.NoError(t, err)
	err = bucket.Load(somekey, func(value []byte) error {
		assert.Equal(t, something, value)
		return nil
	})
	assert.NoError(t, err)
	bucket.Close()
}

func dbPath(t *testing.T) string {
	tmpFile, err := ioutil.TempFile("", "beat.*.db")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	return tmpFile.Name()
}
