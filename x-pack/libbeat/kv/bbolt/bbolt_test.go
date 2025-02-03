package bbolt

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		testCase func() *Bbolt
		expected *Bbolt
	}{
		{
			name: "With no options",
			testCase: func() *Bbolt {
				return New()
			},
			expected: &Bbolt{
				dbPath:     defaultDbPath,
				dbFileMode: defaultDbFileMode,
				bucketName: defaultBucketName,
				db:         nil,
			},
		},
		{
			name: "With options",
			testCase: func() *Bbolt {
				return New(
					WithDbPath("test/path"),
					WithBucketName("test_bucket"),
					WithDbFileMode(0777),
				)
			},
			expected: &Bbolt{
				dbPath:     "test/path",
				dbFileMode: 0777,
				bucketName: "test_bucket",
				db:         nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(tInner *testing.T) {
			boltCache := tt.testCase()
			assert.Equal(tInner, tt.expected, boltCache)
		})
	}
}

func TestGetSet(t *testing.T) {
	tests := []struct {
		name     string
		testCase func(*testing.T, *Bbolt)
	}{
		{
			name: "Simple Set and Get",
			testCase: func(t *testing.T, bolt *Bbolt) {
				err := bolt.Set([]byte("testKey"), []byte("test_value"), 0)
				assert.NoError(t, err)

				val, err := bolt.Get([]byte("testKey"))
				assert.NoError(t, err)
				assert.Equal(t, []byte("test_value"), val)
			},
		},
		{
			name: "Set with expiration",
			testCase: func(t *testing.T, bolt *Bbolt) {
				err := bolt.Set([]byte("testKeyWithExpiration"), []byte("test_value"), 5*time.Second)
				assert.NoError(t, err)

				val, err := bolt.Get([]byte("testKeyWithExpiration"))
				assert.NoError(t, err)
				assert.Equal(t, []byte("test_value"), val)
			},
		},
		{
			name: "Get expired key",
			testCase: func(t *testing.T, bolt *Bbolt) {
				err := bolt.Set([]byte("testKeyWithExpiration2"), []byte("test_value"), time.Nanosecond)
				assert.NoError(t, err)

				time.Sleep(time.Nanosecond) // make sure we wait until key in the cache is expired

				val, err := bolt.Get([]byte("testKeyWithExpiration2"))
				assert.NoError(t, err)
				assert.Nil(t, val)
			},
		},
		{
			name: "Get not existent key",
			testCase: func(t *testing.T, bolt *Bbolt) {
				val, err := bolt.Get([]byte("doesNotExist"))
				assert.NoError(t, err)
				assert.Nil(t, val)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(tInner *testing.T) {
			dbPath := filepath.Join(t.TempDir(), "test.db")

			tInner.Cleanup(func() {
				// Remove test DB file after test is done to not interfere with other tests
				_ = os.Remove(dbPath)
			})

			bolt := &Bbolt{
				dbPath:     dbPath,
				dbFileMode: 0o644,
				bucketName: "test_bucket",
			}

			err := bolt.Connect()
			assert.NoError(t, err)
			defer bolt.Close()

			tt.testCase(t, bolt)
		})
	}
}
