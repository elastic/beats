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

package cptest

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/registry/backend"
)

// BackendFactory is used to instantiate and configure a registry using a
// temporary test path.
type BackendFactory func(testPath string) (backend.Registry, error)

var defaultTempDir string
var keepTmpDir bool

func init() {
	flag.StringVar(&defaultTempDir, "dir", "", "Temporary directory for use by the tests")
	flag.BoolVar(&keepTmpDir, "keep", false, "Keep temporary test directories")
}

// TestBackendCompliance defines a many tests any registry compliant backend must pass.
func TestBackendCompliance(t *testing.T, factory BackendFactory) {
	t.Run("init registry", WithPath(factory, func(t *testing.T, reg *Registry) {
		// none
	}))

	t.Run("access store", WithPath(factory, func(t *testing.T, reg *Registry) {
		store := reg.Access("test1")
		defer store.Close()

		store2 := reg.Access("test2")
		defer store2.Close()
	}))

	t.Run("readonly tx can not write", WithPath(factory, func(t *testing.T, reg *Registry) {
		testReadonlyTx(t, reg)
	}))

	t.Run("remove", withBackend(factory, testRemove))
	t.Run("set-get", withBackend(factory, testSetGet))
	t.Run("update", withBackend(factory, testUpdate))
	t.Run("iteration", withBackend(factory, testIter))
}

func testSetGet(t *testing.T, factory BackendFactory) {
	runWithBools(t, "reopen", func(t *testing.T, reopen bool) {
		t.Run("one entry", WithStore(factory, func(t *testing.T, store *Store) {
			type entry struct{ A int }
			key := makeKey("key")
			value := entry{A: 1}

			store.Set(key, value)
			store.ReopenIf(reopen)

			var actual entry
			store.GetValue(key, &actual)
			assert.Equal(t, value, actual)
		}))

		t.Run("multiple entries", withBools("singletx", func(t *testing.T, singleTx bool) {
			RunWithStore(t, factory, func(store *Store) {
				type entry struct{ A int }
				keys := makeKeys(10)

				if singleTx {
					store.MustUpdate(func(tx *Tx) {
						for i, k := range keys {
							tx.MustSet(k, entry{i})
						}
					})
					store.ReopenIf(reopen)
				} else {
					for i, k := range keys {
						store.Set(k, entry{i})
						store.ReopenIf(reopen)
					}
				}

				// validate
				for i, k := range keys {
					var act entry
					store.GetValue(k, &act)
					assert.Equal(t, entry{i}, act)
				}
			})
		}))

		t.Run("stores are isolated", WithPath(factory, func(t *testing.T, reg *Registry) {
			type entry struct{ A int }
			key := makeKey("key")

			s1 := reg.Access("test1")
			defer s1.Close()

			s2 := reg.Access("test2")
			defer s2.Close()

			// insert into s1 is not visible in s2
			s1.Set(key, entry{1})
			require.False(t, s2.Has(key))
			s1.ReopenIf(reopen)
			s2.ReopenIf(reopen)

			// insert key into s2
			s2.Set(key, entry{2})
			s1.ReopenIf(reopen)
			s2.ReopenIf(reopen)

			// check values are different in s1/s2
			check := func(s *Store, expected int) {
				var actual entry
				s.GetValue(key, &actual)
				require.Equal(t, entry{expected}, actual)
			}
			check(s1, 1)
			check(s2, 2)
		}))

		t.Run("overwrite entry", WithStore(factory, func(t *testing.T, store *Store) {
			type entry1 struct{ A int }
			type entry2 struct{ B int }
			type entry struct {
				A int
				B int
			}

			key := makeKey("key")

			store.Set(key, entry1{1})
			store.ReopenIf(reopen)

			store.MustUpdate(func(tx *Tx) {
				dec := tx.MustGet(key)

				// read original value
				var actual entry
				must(t, dec.Decode(&actual), "faile to decode value")
				assert.Equal(t, entry{A: 1, B: 0}, actual, "invalid value in store")

				tx.MustSet(key, entry2{B: 1})

				// try to get update via decoder
				actual = entry{}
				must(t, dec.Decode(&actual), "failed to decode value")
				assert.Equal(t, entry{A: 0, B: 1}, actual, "decoder state must reflect changes")

				// try to get update via tx
				actual = entry{}
				tx.MustGetValue(key, &actual)
				assert.Equal(t, entry{A: 0, B: 1}, actual, "transaction state must reflect changes")
			})

			store.ReopenIf(reopen)

			// check updated entry is in store
			var actual entry
			store.GetValue(key, &actual)
			assert.Equal(t, entry{A: 0, B: 1}, actual)
		}))

		t.Run("get invalid after tx", withBools("readonly", func(t *testing.T, readonly bool) {
			RunWithStore(t, factory, func(store *Store) {
				type entry struct{ A int }
				key := makeKey("key")
				store.Set(key, entry{1})
				store.ReopenIf(reopen)

				var dec backend.ValueDecoder
				func() {
					tx := store.Begin(readonly)
					defer tx.Close()

					dec = tx.MustGet(key)
				}()

				var tmp entry
				require.NotNil(t, dec)
				assert.Error(t, dec.Decode(&tmp), "expected decoder to fail outside tx")
			})
		}))

		t.Run("decode fails after remove", WithStore(factory, func(t *testing.T, store *Store) {
			type entry struct{ A int }
			key := makeKey("key")
			store.Set(key, entry{1})
			store.ReopenIf(reopen)

			store.MustUpdate(func(tx *Tx) {
				dec := tx.MustGet(key)
				require.NotNil(t, dec)

				tx.Remove(key)

				// validate
				var tmp entry
				assert.Error(t, dec.Decode(&tmp), "expect decode to fail after remove")
			})
		}))
	})

	t.Run("get unknown", withBools("readonly", func(t *testing.T, readonly bool) {
		RunWithStore(t, factory, func(store *Store) {
			tx := store.Begin(readonly)
			defer tx.Close()

			dec, err := tx.Get(makeKey("unknown"))
			assert.NoError(t, err, "get must not error on unknown key")
			assert.Nil(t, dec, "value must be nil")
		})
	}))

	t.Run("failing tx", WithStore(factory, func(t *testing.T, store *Store) {
		type entry struct{ A int }
		key := makeKey("key")

		store.Update(func(tx *Tx) bool {
			tx.MustSet(key, entry{1})
			require.True(t, tx.Has(key))
			return false // fail transaction
		})

		require.False(t, store.Has(key))
	}))
}

func testReadonlyTx(t *testing.T, reg *Registry) {
	store := reg.Access("test")
	defer store.Close()

	tx := store.Begin(true)
	defer tx.Close()

	err := tx.Set(makeKey("key"), map[string]interface{}{})
	if err == nil {
		t.Error("expected write on readonly transaction to fail")
	}
}

func testRemove(t *testing.T, factory BackendFactory) {
	type entry struct{ A int }
	key := makeKey("key")

	t.Run("unknown pair", WithStore(factory, func(t *testing.T, store *Store) {
		tx := store.Begin(false)
		must(t, tx.Remove(key), "be silent about remove of unknown value")
	}))

	runWithBools(t, "reopen", func(t *testing.T, reopen bool) {
		t.Run("stored value", WithStore(factory, func(t *testing.T, store *Store) {
			store.Set(key, entry{1})
			store.ReopenIf(reopen)
			assert.True(t, store.Has(key), "value not in store")

			store.MustUpdate(func(tx *Tx) {
				must(t, tx.Remove(key), "failed to remove entry")
				assert.False(t, tx.Has(key))
			})

			store.ReopenIf(reopen)
			assert.False(t, store.Has(key), "value not removed")
		}))

		t.Run("remove new in tx", WithStore(factory, func(t *testing.T, store *Store) {
			store.MustUpdate(func(tx *Tx) {
				tx.MustSet(key, entry{1})
				require.True(t, tx.Has(key))
				must(t, tx.Remove(key), "failed to remove new entry")
				require.False(t, tx.Has(key))
			})

			store.ReopenIf(reopen)
			require.False(t, store.Has(key))
		}))

		t.Run("remove in failing tx", WithStore(factory, func(t *testing.T, store *Store) {
			store.Set(key, entry{1})

			store.Update(func(tx *Tx) bool {
				assert.True(t, tx.Has(key))
				tx.Remove(key)
				assert.False(t, tx.Has(key))
				return false
			})

			assert.True(t, store.Has(key))
		}))
	})
}

func testUpdate(t *testing.T, factory BackendFactory) {
	type entryA struct{ A int }
	type entryB struct{ B int }
	type entry struct {
		A int
		B int
	}

	runWithBools(t, "reopen", func(t *testing.T, reopen bool) {
		t.Run("unknown entry inserts", WithStore(factory, func(t *testing.T, store *Store) {
			key := makeKey("key")
			store.UpdValue(key, entryA{1})
			store.ReopenIf(reopen)

			assert.True(t, store.Has(key))

			act := entry{A: 2, B: 2}
			store.GetValue(key, &act)
			assert.Equal(t, entry{A: 1, B: 2}, act)
		}))

		t.Run("subset of fields", WithStore(factory, func(t *testing.T, store *Store) {
			key := makeKey("key")
			store.Set(key, entry{A: 2, B: 2})
			store.ReopenIf(reopen)
			store.UpdValue(key, entryA{A: 1})
			store.ReopenIf(reopen)

			act := entry{A: 0, B: 0}
			store.GetValue(key, &act)
			assert.Equal(t, entry{A: 1, B: 2}, act)
		}))

		t.Run("failing tx with unknown", WithStore(factory, func(t *testing.T, store *Store) {
			key := makeKey("key")
			store.Update(func(tx *Tx) bool {
				must(t, tx.Update(key, entryA{3}), "failed to update entry")
				return false
			})
			assert.False(t, store.Has(key))
		}))

		t.Run("failing tx", WithStore(factory, func(t *testing.T, store *Store) {
			key := makeKey("key")
			store.Set(key, entry{A: 1, B: 2})
			store.ReopenIf(reopen)

			store.Update(func(tx *Tx) bool {
				must(t, tx.Update(key, entryA{3}), "failed to update entry")
				return false
			})

			act := entry{A: 0, B: 0}
			store.GetValue(key, &act)
			assert.Equal(t, entry{A: 1, B: 2}, act)
		}))

		t.Run("update removed will insert", WithStore(factory, func(t *testing.T, store *Store) {
			key := makeKey("key")
			store.Set(key, entry{A: 1, B: 2})
			store.ReopenIf(reopen)

			store.MustUpdate(func(tx *Tx) {
				must(t, tx.Remove(key), "failed to remove known key")
				must(t, tx.Update(key, entry{B: 1}), "failed to update removed entry")
			})
			store.ReopenIf(reopen)

			act := entry{A: 0, B: 0}
			store.GetValue(key, &act)
			assert.Equal(t, entry{A: 0, B: 1}, act)
		}))

		t.Run("multiple updates in tx", WithStore(factory, func(t *testing.T, store *Store) {
			key := makeKey("key")
			store.MustUpdate(func(tx *Tx) {
				must(t, tx.Set(key, entryA{A: 1}), "insert failed")
				must(t, tx.Update(key, entryB{B: 2}), "failed to add new field")
				must(t, tx.Update(key, entryA{A: 2}), "failed to update first field")
				must(t, tx.Update(key, entryB{B: 3}), "failed to update second field")
			})
			store.ReopenIf(reopen)

			act := entry{A: 0, B: 0}
			store.GetValue(key, &act)
			assert.Equal(t, entry{A: 2, B: 3}, act)
		}))
	})
}

func testIter(t *testing.T, factory BackendFactory) {
	type entry struct{ A int }

	insert := func(store *Store, keys [][]byte) {
		store.MustUpdate(func(tx *Tx) {
			for i, k := range keys {
				tx.MustSet(k, entry{i})
			}
		})
	}

	runWithBools(t, "readonly", func(t *testing.T, readonly bool) {
		withTx := func(store *Store, readonly bool, fn func(tx *Tx)) {
			tx := store.Begin(readonly)
			defer tx.Close()
			fn(tx)
		}

		t.Run("keys in empty store", WithStore(factory, func(t *testing.T, store *Store) {
			withTx(store, readonly, func(tx *Tx) {
				err := tx.EachKey(false, func(k backend.Key) (bool, error) {
					return true, nil
				})
				assert.NoError(t, err)
			})
		}))

		t.Run("pairs in empty store", WithStore(factory, func(t *testing.T, store *Store) {
			withTx(store, readonly, func(tx *Tx) {
				err := tx.Each(false, func(k backend.Key, val backend.ValueDecoder) (bool, error) {
					return true, nil
				})
				assert.NoError(t, err)
			})
		}))

		t.Run("keys", WithStore(factory, func(t *testing.T, store *Store) {
			keys := makeKeys(10)
			insert(store, keys)

			collected := map[string]int{}
			withTx(store, readonly, func(tx *Tx) {
				err := tx.EachKey(false, func(k backend.Key) (bool, error) {
					var tmp entry
					tx.MustGetValue(k, &tmp)
					collected[string(k)] = tmp.A
					return true, nil
				})
				assert.NoError(t, err)
			})

			for i, k := range keys {
				act, exists := collected[string(k)]
				require.True(t, exists)
				assert.Equal(t, i, act)
			}
		}))

		t.Run("pairs", WithStore(factory, func(t *testing.T, store *Store) {
			keys := makeKeys(10)
			insert(store, keys)

			collected := map[string]int{}
			withTx(store, readonly, func(tx *Tx) {
				err := tx.Each(false, func(k backend.Key, dec backend.ValueDecoder) (bool, error) {
					var tmp entry
					must(t, dec.Decode(&tmp), "failed to decode entry")
					collected[string(k)] = tmp.A
					return true, nil
				})
				assert.NoError(t, err)
			})

			for i, k := range keys {
				act, exists := collected[string(k)]
				require.True(t, exists)
				assert.Equal(t, i, act)
			}
		}))
	})

	t.Run("pairs in active tx", WithStore(factory, func(t *testing.T, store *Store) {
		keys := makeKeys(10)

		collected := map[string]int{}
		store.MustUpdate(func(tx *Tx) {
			for i, k := range keys {
				tx.MustSet(k, entry{i})
			}

			err := tx.Each(false, func(k backend.Key, dec backend.ValueDecoder) (bool, error) {
				var tmp entry
				must(t, dec.Decode(&tmp), "failed to decode entry")
				collected[string(k)] = tmp.A
				return true, nil
			})
			assert.NoError(t, err)
		})

		for i, k := range keys {
			act, exists := collected[string(k)]
			require.True(t, exists, "entry %s is missing", k)
			assert.Equal(t, i, act)
		}
	}))

	t.Run("filled store with inserts in tx", WithStore(factory, func(t *testing.T, store *Store) {
		keys := makeKeys(20)
		sep := len(keys) / 2

		insert(store, keys[:sep])
		collected := map[string]int{}
		store.MustUpdate(func(tx *Tx) {
			for i, k := range keys[sep:] {
				tx.MustSet(k, entry{i + sep})
			}

			err := tx.Each(false, func(k backend.Key, dec backend.ValueDecoder) (bool, error) {
				var tmp entry
				must(t, dec.Decode(&tmp), "failed to decode entry")
				collected[string(k)] = tmp.A
				return true, nil
			})
			assert.NoError(t, err)
		})

		for i, k := range keys {
			act, exists := collected[string(k)]
			require.True(t, exists, "entry %s is missing", k)
			assert.Equal(t, i, act)
		}
	}))

	t.Run("filled store with updates in tx", WithStore(factory, func(t *testing.T, store *Store) {
		keys := makeKeys(20)
		sep := len(keys) / 2

		insert(store, keys)
		collected := map[string]int{}
		store.MustUpdate(func(tx *Tx) {
			for i, k := range keys[sep:] {
				must(t, tx.Update(k, entry{i}), "update failed")
			}

			err := tx.Each(false, func(k backend.Key, dec backend.ValueDecoder) (bool, error) {
				var tmp entry
				must(t, dec.Decode(&tmp), "failed to decode entry")
				collected[string(k)] = tmp.A
				return true, nil
			})
			assert.NoError(t, err)
		})

		for i, k := range keys {
			act, exists := collected[string(k)]
			require.True(t, exists, "entry %s is missing", k)
			if i >= sep {
				i -= sep
			}
			assert.Equal(t, i, act)
		}
	}))
}
