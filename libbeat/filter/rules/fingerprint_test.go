package rules

import (
	"bytes"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

const message = "[Mon Mar 8 05:31:47 2004] [info] [client 64.242.88.10] " +
	"(104)Connection reset by peer: client stopped connection before send " +
	"body completed"

func newTestFingerprint(t testing.TB, hash string, opts ...map[string]interface{}) *Fingerprint {
	c, err := common.NewConfigFrom(map[string]interface{}{"hash": hash})
	if err != nil {
		t.Fatal(err)
	}

	if len(opts) == 1 {
		err := c.Merge(opts[0])
		if err != nil {
			t.Fatal(err)
		}
	}

	f, err := newFingerprint(*c)
	if err != nil {
		t.Fatal(err)
	}
	return f.(*Fingerprint)
}

func TestFingerprintHashes(t *testing.T) {
	var tests = []struct {
		hash        string
		fingerprint string
	}{
		{"sha1", "fe7b2aede2119f5508f466209a26a863d405c1ee"},
		{"sha256", "5c2736f2a1b8ec165ffe3e904b4171ad7581db825f492bca7bdfa0cca4e5630f"},
		{"sha512", "3142c7ff002141dba401d5cae154f4c534d5e3b8403699398c3e1dd20bcad764accccfab108188f14b01e11072c3a705d8d355473ebbea576f008920d6953b4e"},
		{"md5", "ffc2c1d636ac17a38860df03350be0e4"},
	}

	for _, testcase := range tests {
		f := newTestFingerprint(t, testcase.hash)
		event := common.MapStr{"message": message}
		event, err := f.Filter(event)
		if assert.NoError(t, err) {
			assert.Equal(t, testcase.fingerprint, event["id"])
		}
	}
}

func TestFingerprintFieldConcat(t *testing.T) {
	f := newTestFingerprint(t, "sha1", map[string]interface{}{
		"fields": []string{"@timestamp", "record_number", "beat.host", "message"},
	})
	event := common.MapStr{
		"@timestamp":    common.Time(time.Unix(1091067890, 0)),
		"record_number": 1888399992,
		"beat": common.MapStr{
			"host": "example",
		},
		"message": message,
	}

	event, err := f.Filter(event)
	if assert.NoError(t, err) {
		assert.Equal(t, "ee89e405f814a308440c20f11adcc36e9e51c393", event["id"])
	}
}

func TestFingerprintMissingField(t *testing.T) {
	f := newTestFingerprint(t, "sha1", map[string]interface{}{
		"fields": []string{"other"},
	})
	event := common.MapStr{"message": message}

	event, err := f.Filter(event)
	assert.Error(t, err)
	assert.NotNil(t, event)
}

func TestFingerprintString(t *testing.T) {
	f := newTestFingerprint(t, "sha1")
	assert.Equal(t, "fingerprint=[fields=message, hash=sha1, target=id]", f.String())
}

func TestWriteValue(t *testing.T) {
	var tests = []struct {
		in  interface{}
		out string
	}{
		{nil, "nil"},
		{true, "true"},
		{8, "8"},
		{uint(10), "10"},
		{18.123, "18.123"},
		{"hello", "hello"},
	}

	b := new(bytes.Buffer)
	for _, testcase := range tests {
		b.Reset()
		writeValue(b, testcase.in)
		assert.Equal(t, testcase.out, b.String())

		b.Reset()
		writeValue(b, &testcase.in)
		assert.Equal(t, testcase.out, b.String())
	}
}

func BenchmarkFingerprintFilterSHA1(b *testing.B) {
	f := newTestFingerprint(b, "sha1")
	event := common.MapStr{"message": message}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Filter(event)
	}
}

func BenchmarkFingerprintFilterSHA256(b *testing.B) {
	f := newTestFingerprint(b, "sha256")
	event := common.MapStr{"message": message}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Filter(event)
	}
}

func BenchmarkFingerprintFilterSHA512(b *testing.B) {
	f := newTestFingerprint(b, "sha512")
	event := common.MapStr{"message": message}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Filter(event)
	}
}

func BenchmarkFingerprintFilterMD5(b *testing.B) {
	f := newTestFingerprint(b, "md5")
	event := common.MapStr{"message": message}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Filter(event)
	}
}
