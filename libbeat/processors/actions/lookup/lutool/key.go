package lutool

import (
	"hash"

	"github.com/elastic/beats/libbeat/common"
)

type Key struct {
	keys []string
}

type KeyBuilder interface {
	ExtractKey(common.MapStr) (Key, bool)
}

type KeyBuilderFunc func(common.MapStr) (Key, bool)

func (k *Key) Equals(o *Key) bool {
	if len(k.keys) != len(k.keys) {
		return false
	}

	for i := range k.keys {
		if k.keys[i] != o.keys[i] {
			return false
		}
	}
	return true
}

func (k *Key) Hash(h hash.Hash64) error {
	for i, s := range k.keys {
		var err error

		_, err = h.Write([]byte{byte(i), byte(i >> 8)})
		if err != nil {
			return err
		}

		_, err = h.Write([]byte(s))
		if err != nil {
			return err
		}
	}
	return nil
}

func MakeKeyBuilder(fields []string) (KeyBuilder, error) {
	fn, err := makeFieldsCollector(fields)
	if err != nil {
		return nil, err
	}

	return KeyBuilderFunc(func(event common.MapStr) (Key, bool) {
		values, ok := fn(event)
		return Key{values}, ok
	}), nil
}
