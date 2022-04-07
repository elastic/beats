// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package awss3

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/elastic/beats/v8/libbeat/logp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionScriptParams(t *testing.T) {
	logp.TestingSetup()

	t.Run("register method is optional", func(t *testing.T) {
		_, err := newScriptFromConfig(log, &scriptConfig{Source: header + footer})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("register required for params", func(t *testing.T) {
		_, err := newScriptFromConfig(log, &scriptConfig{
			Source: header + footer, Params: map[string]interface{}{
				"p1": 42,
			},
		})
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "params were provided")
		}
	})

	t.Run("register params", func(t *testing.T) {
		const script = `
			function register(params) {
				if (params["p1"] !== 42) {
					throw "invalid p1";
				}
			}

			function parse(n) {}
		`
		_, err := newScriptFromConfig(log, &scriptConfig{
			Source: script,
			Params: map[string]interface{}{
				"p1": 42,
			},
		})
		assert.NoError(t, err)
	})
}

func TestSessionTestFunction(t *testing.T) {
	logp.TestingSetup()

	const script = `
		var fail = false;

		function register(params) {
			fail = params["fail"];
		}

		function parse(n) {
			if (fail) {
				throw "intentional failure";
			}
			var m = JSON.parse(n);
			var e = new S3EventV2();
			e.SetS3ObjectKey(m["hello"]);
			return [e];
		}

		function test() {
			var n = "{\"hello\": \"earth\"}";
			var evts = parse(n);

			if (evts[0].S3.Object.Key !== "earth") {
				throw "invalid key value";
 			}
		}
	`

	t.Run("test method is optional", func(t *testing.T) {
		_, err := newScriptFromConfig(log, &scriptConfig{
			Source: header + footer,
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test success", func(t *testing.T) {
		_, err := newScriptFromConfig(log, &scriptConfig{
			Source: script,
			Params: map[string]interface{}{
				"fail": false,
			},
		})
		assert.NoError(t, err)
	})

	t.Run("test failure", func(t *testing.T) {
		_, err := newScriptFromConfig(log, &scriptConfig{
			Source: script,
			Params: map[string]interface{}{
				"fail": true,
			},
		})
		assert.Error(t, err)
	})
}

func TestSessionTimeout(t *testing.T) {
	logp.TestingSetup()

	const runawayLoop = `
		var m = JSON.parse(n);
		while (!m.stop) {
			m.hello = "world";
		}
    `

	p, err := newScriptFromConfig(log, &scriptConfig{
		Source:  header + runawayLoop + footer,
		Timeout: 100 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	n := `{"stop": false}`

	// Execute and expect a timeout.
	_, err = p.run(n)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), timeoutError)
	}

	// Verify that any internal runtime interrupt state has been cleared.
	n = `{"stop": true}`
	_, err = p.run(n)
	assert.NoError(t, err)
}

func TestSessionParallel(t *testing.T) {
	logp.TestingSetup()

	const script = `
		var m = JSON.parse(n);
		var evt = new S3EventV2();
		evt.SetS3ObjectKey(m.hello.world);
		return [evt];
    `

	p, err := newScriptFromConfig(log, &scriptConfig{
		Source: header + script + footer,
	})
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 10
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				n := `{"hello":{"world": "hello"}}`
				evts, err := p.run(n)
				require.NoError(t, err)
				require.Equal(t, 1, len(evts))
				assert.Equal(t, "hello", evts[0].S3.Object.Key)
			}
		}()
	}

	time.AfterFunc(time.Second, cancel)
	wg.Wait()
}

func TestCreateS3EventsFromNotification(t *testing.T) {
	logp.TestingSetup()

	n := `{
		"cid":        "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		"timestamp":  1492726639222,
		"fileCount":  4,
		"totalSize":  349986221,
		"bucket":     "bucketNNNN",
		"pathPrefix": "logs/aaaa-bbbb-cccc-dddd-eeee-ffff",
		"files": [
			{
				"path":     "logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00000.gz",
				"size":     90506437,
				"checksum": "ffffffffffffffffffff"
			},
			{
				"path":     "logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00001.gz",
				"size":     86467594,
				"checksum": "ffffffffffffffffffff"
			},
			{
				"path":     "logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00002.gz",
				"size":     83893710,
				"checksum": "ffffffffffffffffffff"
			},
			{
				"path":     "logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00003.gz",
				"size":     89118480,
				"checksum": "ffffffffffffffffffff"
			}
		]
	}`

	const script = `
	function parse(n) {
		var m = JSON.parse(n);
		var evts = [];
		var files = m.files;
		var bucket = m.bucket;

		if (!Array.isArray(files) || (files.length == 0) || bucket == null || bucket == "") {
			return evts;
		}

		files.forEach(function(f){
			var evt = new S3EventV2();
			evt.SetS3BucketName(bucket);
			evt.SetS3ObjectKey(f.path);
			evts.push(evt);
		});

		return evts;
	}
`
	s, err := newScriptFromConfig(log, &scriptConfig{Source: script})
	require.NoError(t, err)

	evts, err := s.run(n)
	require.NoError(t, err)
	require.Equal(t, 4, len(evts))

	const expectedBucket = "bucketNNNN"
	expectedObjectKeys := []string{
		"logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00000.gz",
		"logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00001.gz",
		"logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00002.gz",
		"logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00003.gz",
	}

	for i, e := range expectedObjectKeys {
		assert.Equal(t, expectedBucket, evts[i].S3.Bucket.Name)
		assert.Equal(t, e, evts[i].S3.Object.Key)
	}
}

func TestParseXML(t *testing.T) {
	logp.TestingSetup()

	n := `<record>
	<bucket>bucketNNNN</bucket>
	<files>
		<file><path>logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00000.gz</path></file>
		<file><path>logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00001.gz</path></file>
	</files>
	</record>`

	const script = `
	function parse(n) {
		var dec = new XMLDecoder(n);
		var m = dec.Decode();
		var evts = [];
		var files = m.record.files.file;
		var bucket = m.record.bucket;

		if (!Array.isArray(files) || (files.length == 0) || bucket == null || bucket == "") {
			return evts;
		}

		files.forEach(function(f){
			var evt = new S3EventV2();
			evt.SetS3BucketName(bucket);
			evt.SetS3ObjectKey(f.path);
			evts.push(evt);
		});

		return evts;
	}
`
	s, err := newScriptFromConfig(log, &scriptConfig{Source: script})
	require.NoError(t, err)

	evts, err := s.run(n)
	require.NoError(t, err)
	require.Equal(t, 2, len(evts))

	const expectedBucket = "bucketNNNN"
	expectedObjectKeys := []string{
		"logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00000.gz",
		"logs/aaaa-bbbb-cccc-dddd-eeee-ffff/part-00001.gz",
	}

	for i, e := range expectedObjectKeys {
		assert.Equal(t, expectedBucket, evts[i].S3.Bucket.Name)
		assert.Equal(t, e, evts[i].S3.Object.Key)
	}
}
