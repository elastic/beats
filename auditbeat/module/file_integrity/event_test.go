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
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"os/user"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

var testEventTime = time.Now().UTC()

func testEvent() *Event {
	return &Event{
		Timestamp: testEventTime,
		Path:      "/home/user/file.txt",
		Source:    SourceScan,
		Action:    ConfigChange,
		Info: &Metadata{
			Type:   FileType,
			Inode:  123,
			UID:    500,
			GID:    500,
			Mode:   0o600,
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
		e.Info.Mode = 0o644

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
	f, err := os.CreateTemp(t.TempDir(), "input.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	const data = "hello world!\n"
	const dataLen = uint64(len(data))
	_, err = f.WriteString(data)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Sync()
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

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

		hashes, size, err := hashFile(f.Name(), dataLen, validHashes...)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, dataLen, size)
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
		hashes, size, err := hashFile("anyfile.txt", 1234)
		assert.Nil(t, hashes)
		assert.NoError(t, err)
		assert.Zero(t, size)
	})

	t.Run("invalid hash", func(t *testing.T) {
		hashes, size, err := hashFile("anyfile.txt", 1234, "md4")
		assert.Nil(t, hashes)
		assert.Error(t, err)
		assert.Zero(t, size)
	})

	t.Run("invalid file", func(t *testing.T) {
		hashes, size, err := hashFile("anyfile.txt", 1234, "md5")
		assert.Nil(t, hashes)
		assert.Error(t, err)
		assert.Zero(t, size)
	})

	t.Run("size over hash limit", func(t *testing.T) {
		hashes, size, err := hashFile(f.Name(), dataLen-1, SHA1)
		assert.Nil(t, hashes)
		assert.Zero(t, size)
		assert.NoError(t, err)
	})
	t.Run("size at hash limit", func(t *testing.T) {
		hashes, size, err := hashFile(f.Name(), dataLen, SHA1)
		assert.NotNil(t, hashes)
		assert.Equal(t, dataLen, size)
		assert.NoError(t, err)
	})
	t.Run("size below hash limit", func(t *testing.T) {
		hashes, size, err := hashFile(f.Name(), dataLen+1, SHA1)
		assert.NotNil(t, hashes)
		assert.Equal(t, dataLen, size)
		assert.NoError(t, err)
	})
	t.Run("no size limit", func(t *testing.T) {
		hashes, size, err := hashFile(f.Name(), math.MaxInt64, SHA1)
		assert.NotNil(t, hashes)
		assert.Equal(t, dataLen, size)
		assert.NoError(t, err)
	})
}

func TestNewEventFromFileInfoHash(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "input.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	const data = "hello world!\n"
	const dataLen = uint64(len(data))
	_, err = f.WriteString(data)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Sync()
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	info, err := os.Stat(f.Name())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("file stays the same", func(t *testing.T) {
		ev := NewEventFromFileInfo(f.Name(), info, nil, Updated, SourceFSNotify, MaxValidFileSizeLimit, []HashType{SHA1}, nil)
		if !assert.NotNil(t, ev) {
			t.Fatal("nil event")
		}
		assert.Equal(t, dataLen, ev.Info.Size)
		assert.NotNil(t, ev.Hashes)
		digest := Digest(mustDecodeHex("f951b101989b2c3b7471710b4e78fc4dbdfa0ca6"))
		assert.Equal(t, digest, ev.Hashes[SHA1])
	})
	t.Run("file grows before hashing", func(t *testing.T) {
		_, err = f.WriteString(data)
		if err != nil {
			t.Fatal(err)
		}
		err = f.Sync()
		if err != nil {
			t.Fatal(err)
		}
		ev := NewEventFromFileInfo(f.Name(), info, nil, Updated, SourceFSNotify, MaxValidFileSizeLimit, []HashType{SHA1}, nil)
		if !assert.NotNil(t, ev) {
			t.Fatal("nil event")
		}
		assert.Equal(t, dataLen*2, ev.Info.Size)
		assert.NotNil(t, ev.Hashes)
		digest := Digest(mustDecodeHex("62e8a0ef77ed7596347a065cae28a860f87e382f"))
		assert.Equal(t, digest, ev.Hashes[SHA1])
	})
	t.Run("file shrinks before hashing", func(t *testing.T) {
		err = f.Truncate(0)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		err = f.Sync()
		if err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, err)
		ev := NewEventFromFileInfo(f.Name(), info, nil, Updated, SourceFSNotify, MaxValidFileSizeLimit, []HashType{SHA1}, nil)
		if !assert.NotNil(t, ev) {
			t.Fatal("nil event")
		}
		assert.Zero(t, ev.Info.Size)
		assert.NotNil(t, ev.Hashes)
		digest := Digest(mustDecodeHex("da39a3ee5e6b4b0d3255bfef95601890afd80709"))
		assert.Equal(t, digest, ev.Hashes[SHA1])
	})
}

func BenchmarkHashFile(b *testing.B) {
	f, err := os.CreateTemp(b.TempDir(), "hash")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())

	zeros := make([]byte, 100)
	const iterations = 1024 * 1024 // 100 MiB
	for i := 0; i < iterations; i++ {
		if _, err = f.Write(zeros); err != nil {
			b.Fatal(err)
		}
	}
	size := uint64(iterations * len(zeros))
	b.Logf("file size: %v bytes", size)
	err = f.Sync()
	if err != nil {
		b.Fatal(err)
	}
	f.Close()
	b.ResetTimer()

	for _, hashType := range validHashes {
		b.Run(string(hashType), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, nbytes, err := hashFile(f.Name(), size+1, hashType)
				if err != nil {
					b.Fatal(err)
				}
				assert.Equal(b, size, nbytes)
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
		assertHasKey(t, fields, "event.kind")
		assertHasKey(t, fields, "event.category")
		assertHasKey(t, fields, "event.type")

		assertHasKey(t, fields, "file.path")
		if assertHasKey(t, fields, "file.extension") {
			ext, err := fields.GetValue("file.extension")
			require.NoError(t, err)
			assert.Equal(t, ext, "txt")
		}
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

		assertHasKey(t, fields, "file.hash.sha1")
		assertHasKey(t, fields, "file.hash.sha256")
	})
	if runtime.GOOS == "windows" {
		t.Run("drive letter", func(t *testing.T) {
			e := testEvent()
			e.Path = "c:\\Documents"
			fields := buildMetricbeatEvent(e, false).MetricSetFields
			value, err := fields.GetValue("file.drive_letter")
			assert.NoError(t, err)
			assert.Equal(t, "C", value)
		})
		t.Run("no drive letter", func(t *testing.T) {
			e := testEvent()
			e.Path = "\\\\remote\\Documents"
			fields := buildMetricbeatEvent(e, false).MetricSetFields
			_, err := fields.GetValue("file.drive_letter")
			assert.Error(t, err)
		})
	}
	t.Run("ecs categorization", func(t *testing.T) {
		e := testEvent()
		e.Action = ConfigChange
		fields := buildMetricbeatEvent(e, false).MetricSetFields
		types, err := fields.GetValue("event.type")
		if err != nil {
			t.Fatal(err)
		}
		ecsTypes, ok := types.([]string)
		assert.True(t, ok)
		assert.Equal(t, []string{"change"}, ecsTypes)

		e.Action = Action(Created | Updated | Deleted)
		fields = buildMetricbeatEvent(e, false).MetricSetFields
		types, err = fields.GetValue("event.type")
		if err != nil {
			t.Fatal(err)
		}
		ecsTypes, ok = types.([]string)
		assert.True(t, ok)
		assert.Equal(t, []string{"change", "creation", "deletion"}, ecsTypes)
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
		panic(fmt.Errorf("invalid hex value: %w", err))
	}
	return data
}

func assertHasKey(t testing.TB, m mapstr.M, key string) bool {
	t.Helper()
	found, err := m.HasKey(key)
	if err != nil || !found {
		t.Errorf("key %v not found: %v", key, err)
		return false
	}
	return true
}

func TestACLText(t *testing.T) {
	// The xattr package returns raw bytes, but command line tools such as getfattr
	// return a base64-encoded format, so use that here to make test validation
	// easier.
	//
	// Depending on the system we are running this test on, we may or may not
	// have a username associated with the user's UID in the xattr string, so
	// dynamically determine the username here.
	tests := []struct {
		encoded string
		want    []string
	}{
		0: {
			encoded: "0sAgAAAAEABgD/////AgAGAG8AAAAEAAQA/////xAABgD/////IAAEAP////8=",
			want:    []string{"user::rw-", "user:" + userNameOrUID("111") + ":rw-", "group::r--", "mask::rw-", "other::r--"},
		},
		1: { // Encoded string from https://www.bityard.org/wiki/tech/os/linux/xattrs.
			encoded: "0sAgAAAAEABgD/////AgAHAHwAAAAEAAQA/////xAABwD/////IAAEAP////8=",
			want:    []string{"user::rw-", "user:" + userNameOrUID("124") + ":rwx", "group::r--", "mask::rwx", "other::r--"},
		},
	}
	for i, test := range tests {
		b, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(test.encoded, "0s"))
		if err != nil {
			t.Errorf("invalid test: unexpected base64 encoding error for test %d: %v", i, err)
			continue
		}
		got, err := aclText(b)
		if err != nil {
			t.Errorf("unexpected error for test %d: %v", i, err)
			continue
		}
		if !reflect.DeepEqual(got, test.want) {
			t.Errorf("unexpected result for test %d:\ngot: %#v\nwant:%#v", i, got, test.want)
		}
	}
}

func userNameOrUID(uid string) string {
	u, err := user.LookupId(uid)
	if err != nil {
		return uid
	}
	return u.Username
}
