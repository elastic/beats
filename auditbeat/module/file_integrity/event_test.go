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

package file_integrity

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
)

var testEventTime = time.Now().UTC()

func testEvent() *Event {
	return &Event{
		Timestamp: testEventTime,
		Path:      "/home/user",
		Source:    SourceScan,
		Action:    ConfigChange,
		Info: &Metadata{
			Type:   FileType,
			Inode:  123,
			UID:    500,
			GID:    500,
			Mode:   0600,
			CTime:  testEventTime,
			MTime:  testEventTime,
			SetGID: true,
		},
		Hashes: map[HashType]Digest{
			SHA1:   mustDecodeHex("abcd"),
			SHA256: mustDecodeHex("1234"),
		},
	}
}

func TestDiffEvents(t *testing.T) {
	t.Run("nil values", func(t *testing.T) {
		_, changed := diffEvents(nil, nil)
		assert.False(t, changed)
	})

	t.Run("no change", func(t *testing.T) {
		e := testEvent()
		_, changed := diffEvents(e, e)
		assert.False(t, changed)
	})

	t.Run("new file", func(t *testing.T) {
		action, changed := diffEvents(nil, testEvent())
		assert.True(t, changed)
		assert.EqualValues(t, Created, action)
	})

	t.Run("deleted", func(t *testing.T) {
		action, changed := diffEvents(testEvent(), nil)
		assert.True(t, changed)
		assert.EqualValues(t, Deleted, action)
	})

	t.Run("moved", func(t *testing.T) {
		e := testEvent()
		e.Path += "_new"

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, Moved, action)
	})

	t.Run("updated metadata", func(t *testing.T) {
		e := testEvent()
		e.Info.Mode = 0644

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, AttributesModified, action, "action: %v", action)
	})

	t.Run("missing metadata", func(t *testing.T) {
		e := testEvent()
		e.Info = nil

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, AttributesModified, action)
	})

	t.Run("more hashes", func(t *testing.T) {
		e := testEvent()
		e.Hashes["md5"] = mustDecodeHex("5678")

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, ConfigChange, action)
	})

	t.Run("subset of hashes", func(t *testing.T) {
		e := testEvent()
		delete(e.Hashes, "sha256")

		action, changed := diffEvents(testEvent(), e)
		assert.False(t, changed)
		assert.Zero(t, action)
	})

	t.Run("different hash values", func(t *testing.T) {
		e := testEvent()
		e.Hashes = map[HashType]Digest{
			SHA1:   mustDecodeHex("ef"),
			SHA256: mustDecodeHex("1234"),
		}

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, Updated, action)
	})

	t.Run("updated hashes and metadata", func(t *testing.T) {
		e := testEvent()
		e.Hashes = map[HashType]Digest{
			SHA1:   mustDecodeHex("ef"),
			SHA256: mustDecodeHex("1234"),
		}
		e.Info.MTime = time.Now()

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, Updated|AttributesModified, action)
	})

	t.Run("updated setuid field", func(t *testing.T) {
		e := testEvent()
		e.Info.SetUID = true

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, AttributesModified, action, "action: %v", action)
	})

	t.Run("updated setgid field", func(t *testing.T) {
		e := testEvent()
		e.Info.SetGID = false

		action, changed := diffEvents(testEvent(), e)
		assert.True(t, changed)
		assert.EqualValues(t, AttributesModified, action, "action: %v", action)
	})
}

func TestHashFile(t *testing.T) {
	t.Run("valid hashes", func(t *testing.T) {
		// Computed externally.
		expectedHashes := map[HashType]Digest{
			BLAKE2B_256: mustDecodeHex("0f0cc1f0ea4ef962d6a150ae0b77bc320b57ed24e1609b933fa2274484f59145"),
			BLAKE2B_384: mustDecodeHex("b819d90f648da6effff2393acb1884d2638642b3524c329832c073c9364149fcdedb522914ef9c2c92f007a42366139a"),
			BLAKE2B_512: mustDecodeHex("fc13029e8a5ce67ad5a70f0cc659a4b30df9d791b125835e434606c6127ee37ebbc8b216389682ddfa84380789db09f2535d2a9837454414ea3ff00ec0801150"),
			MD5:         mustDecodeHex("c897d1410af8f2c74fba11b1db511e9e"),
			SHA1:        mustDecodeHex("f951b101989b2c3b7471710b4e78fc4dbdfa0ca6"),
			SHA224:      mustDecodeHex("d301812e62eec9b1e68c0b861e62f374e0d77e8365f5ddd6cccc8693"),
			SHA256:      mustDecodeHex("ecf701f727d9e2d77c4aa49ac6fbbcc997278aca010bddeeb961c10cf54d435a"),
			SHA384:      mustDecodeHex("ec8d147738b2e4bf6f5c5ac50a9a7593fb1ee2de01474d6f8a6c7fdb7ac945580772a5225a4c7251a7c0697acb7b8405"),
			SHA512:      mustDecodeHex("f5408390735bf3ef0bb8aaf66eff4f8ca716093d2fec50996b479b3527e5112e3ea3b403e9e62c72155ac1e08a49b476f43ab621e1a5fc2bbb0559d8258a614d"),
			SHA512_224:  mustDecodeHex("fde054253f43a95559f1b6eeb8e2edba4124957b43b85d7fcb4d20d5"),
			SHA512_256:  mustDecodeHex("3380f6a625aac19cbdddc598ab07aea195bae000f8d4c8cd6bb8870ac25df15d"),
			SHA3_224:    mustDecodeHex("62e3515dae95bbd0e105bee840b7dc3b47f6d6bc772c259dbc0da31a"),
			SHA3_256:    mustDecodeHex("3cb5385a2987ca45888d7877fbcf92b4854f7155ae19c96cecc7ea1300c6f5a4"),
			SHA3_384:    mustDecodeHex("f19539818b4f29fa0ee599db4113fd81b77cd1119682e6d799a052849d2e40ef0dad84bc947ba2dee742d9731f1b9e9b"),
			SHA3_512:    mustDecodeHex("f0a2c0f9090c1fd6dedf211192e36a6668d2b3c7f57a35419acb1c4fc7dfffc267bbcd90f5f38676caddcab652f6aacd1ed4e0ad0a8e1e4b98f890b62b6c7c5c"),
			XXH64:       mustDecodeHex("d3e8573b7abf279a"),
		}

		f, err := ioutil.TempFile("", "input.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		f.WriteString("hello world!\n")
		f.Sync()
		f.Close()

		hashes, err := hashFile(f.Name(), validHashes...)
		if err != nil {
			t.Fatal(err)
		}

		for _, hashType := range validHashes {
			if hash, found := hashes[hashType]; !found {
				t.Errorf("%v not found", hashType)
			} else {
				delete(hashes, hashType)
				expected, ok := expectedHashes[hashType]
				if !ok {
					t.Fatalf("hash type not found in expected hashes: %v", hashType)
				}
				if !bytes.Equal(expected, hash) {
					t.Errorf("%v hash incorrect, got: %v, want: %v", hashType,
						hex.EncodeToString(hash), hex.EncodeToString(expected))
				}
			}
		}

		assert.Len(t, hashes, 0)
	})

	t.Run("no hashes", func(t *testing.T) {
		hashes, err := hashFile("anyfile.txt")
		assert.Nil(t, hashes)
		assert.NoError(t, err)
	})

	t.Run("invalid hash", func(t *testing.T) {
		hashes, err := hashFile("anyfile.txt", "md4")
		assert.Nil(t, hashes)
		assert.Error(t, err)
	})

	t.Run("invalid file", func(t *testing.T) {
		hashes, err := hashFile("anyfile.txt", "md5")
		assert.Nil(t, hashes)
		assert.Error(t, err)
	})
}

func BenchmarkHashFile(b *testing.B) {
	f, err := ioutil.TempFile("", "hash")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())

	zeros := make([]byte, 100)
	iterations := 1024 * 1024 // 100 MiB
	for i := 0; i < iterations; i++ {
		if _, err = f.Write(zeros); err != nil {
			b.Fatal(err)
		}
	}
	b.Logf("file size: %v bytes", len(zeros)*iterations)
	f.Sync()
	f.Close()
	b.ResetTimer()

	for _, hashType := range validHashes {
		b.Run(string(hashType), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err = hashFile(f.Name(), hashType)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestBuildEvent(t *testing.T) {
	t.Run("all fields", func(t *testing.T) {
		e := testEvent()
		e.TargetPath = "link_target"
		e.Info.SID = "S-123-4"
		e.Info.Owner = "beats"
		e.Info.Group = "staff"
		e.Info.SetUID = true
		e.Info.Origin = []string{"google.com"}

		fields := buildMetricbeatEvent(e, false).MetricSetFields
		assert.Equal(t, testEventTime, e.Timestamp)

		assertHasKey(t, fields, "event.action")

		assertHasKey(t, fields, "file.path")
		assertHasKey(t, fields, "file.target_path")
		assertHasKey(t, fields, "file.inode")
		assertHasKey(t, fields, "file.uid")
		assertHasKey(t, fields, "file.owner")
		assertHasKey(t, fields, "file.group")
		assertHasKey(t, fields, "file.size")
		assertHasKey(t, fields, "file.mtime")
		assertHasKey(t, fields, "file.ctime")
		assertHasKey(t, fields, "file.type")
		assertHasKey(t, fields, "file.setuid")
		assertHasKey(t, fields, "file.setgid")
		assertHasKey(t, fields, "file.origin")
		if runtime.GOOS != "windows" {
			assertHasKey(t, fields, "file.gid")
			assertHasKey(t, fields, "file.mode")
		}

		assertHasKey(t, fields, "hash.sha1")
		assertHasKey(t, fields, "hash.sha256")
	})
	t.Run("no setuid/setgid", func(t *testing.T) {
		e := testEvent()
		e.Info.SetGID = false
		fields := buildMetricbeatEvent(e, false).MetricSetFields
		_, err := fields.GetValue("file.setuid")
		assert.Error(t, err)
		_, err = fields.GetValue("file.setgid")
		assert.Error(t, err)
	})
	t.Run("setgid set", func(t *testing.T) {
		e := testEvent()
		fields := buildMetricbeatEvent(e, false).MetricSetFields
		_, err := fields.GetValue("file.setuid")
		assert.Error(t, err)

		setgid, err := fields.GetValue("file.setgid")
		if err != nil {
			t.Fatal(err)
		}
		flag, ok := setgid.(bool)
		assert.True(t, ok)
		assert.True(t, flag)
	})
	t.Run("setuid set", func(t *testing.T) {
		e := testEvent()
		e.Info.SetUID = true
		e.Info.SetGID = false
		fields := buildMetricbeatEvent(e, false).MetricSetFields
		_, err := fields.GetValue("file.setgid")
		assert.Error(t, err)

		setgid, err := fields.GetValue("file.setuid")
		if err != nil {
			t.Fatal(err)
		}
		flag, ok := setgid.(bool)
		assert.True(t, ok)
		assert.True(t, flag)
	})
	t.Run("setuid and setgid set", func(t *testing.T) {
		e := testEvent()
		e.Info.SetUID = true
		fields := buildMetricbeatEvent(e, false).MetricSetFields
		setuid, err := fields.GetValue("file.setgid")
		if err != nil {
			t.Fatal(err)
		}
		flag, ok := setuid.(bool)
		assert.True(t, ok)
		assert.True(t, flag)

		setgid, err := fields.GetValue("file.setuid")
		if err != nil {
			t.Fatal(err)
		}
		flag, ok = setgid.(bool)
		assert.True(t, ok)
		assert.True(t, flag)
	})
}

func mustDecodeHex(v string) []byte {
	data, err := hex.DecodeString(v)
	if err != nil {
		panic(fmt.Errorf("invalid hex value: %v", err))
	}
	return data
}

func assertHasKey(t testing.TB, m common.MapStr, key string) {
	t.Helper()
	found, err := m.HasKey(key)
	if err != nil || !found {
		t.Errorf("key %v not found: %v", key, err)
	}
}
