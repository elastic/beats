package lutool

import (
	"hash/fnv"
	"testing"

	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

func TestKey(t *testing.T) {
	tests := []struct {
		title  string
		keys   []string
		e1, e2 common.MapStr
		equals bool
	}{
		{
			"same compound key",
			[]string{"a", "b"},
			common.MapStr{"a": 1, "b": 2, "c": true},
			common.MapStr{"a": 1, "b": 2, "c": false},
			true,
		},
		{
			"different compound key",
			[]string{"a", "b"},
			common.MapStr{"a": 1, "b": 2, "c": true},
			common.MapStr{"a": 2, "b": 1, "c": false},
			false,
		},
		{
			"different but similar keys",
			[]string{"a", "b"},
			common.MapStr{"a": "abc", "b": "d"},
			common.MapStr{"a": "ab", "b": "cd"},
			false,
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		kb, err := MakeKeyBuilder(test.keys)
		if err != nil {
			t.Error(err)
			continue
		}

		k1, b1 := kb.ExtractKey(test.e1)
		k2, b2 := kb.ExtractKey(test.e2)
		if !b1 || !b2 {
			assert.True(t, b1)
			assert.True(t, b2)
			continue
		}

		hash := fnv.New64()
		err = k1.Hash(hash)
		assert.NoError(t, err)
		h1 := hash.Sum64()

		hash.Reset()
		err = k2.Hash(hash)
		assert.NoError(t, err)
		h2 := hash.Sum64()

		t.Log(h1)
		t.Log(h2)

		if test.equals {
			assert.Equal(t, h1, h2)
			assert.True(t, k1.Equals(&k2))
			assert.True(t, k2.Equals(&k1))
		} else {
			assert.NotEqual(t, h1, h2)
			assert.False(t, k1.Equals(&k2))
			assert.False(t, k2.Equals(&k1))
		}
	}
}
