// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kvstore

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/entcollect"
)

var _ entcollect.Store = (*EntcollectStore)(nil)

// EntcollectStore wraps a *Transaction (single bbolt bucket) to
// satisfy the entcollect.Store interface.
type EntcollectStore struct {
	tx     *Transaction
	bucket []byte
}

// NewEntcollectStore returns an entcollect.Store backed by tx,
// scoping all operations to the named bucket.
func NewEntcollectStore(tx *Transaction, bucket string) *EntcollectStore {
	return &EntcollectStore{
		tx:     tx,
		bucket: []byte(bucket),
	}
}

func (s *EntcollectStore) Get(key string, dst any) error {
	err := s.tx.Get(s.bucket, []byte(key), dst)
	if err != nil {
		if errors.Is(err, ErrBucketNotFound) || errors.Is(err, ErrKeyNotFound) {
			return fmt.Errorf("entcollect store get %q: %w", key, entcollect.ErrKeyNotFound)
		}
		return fmt.Errorf("entcollect store get %q: %w", key, err)
	}
	return nil
}

func (s *EntcollectStore) Set(key string, value any) error {
	err := s.tx.Set(s.bucket, []byte(key), value)
	if err != nil {
		return fmt.Errorf("entcollect store set %q: %w", key, err)
	}
	return nil
}

func (s *EntcollectStore) Delete(key string) error {
	err := s.tx.Delete(s.bucket, []byte(key))
	if err != nil {
		return fmt.Errorf("entcollect store delete %q: %w", key, err)
	}
	return nil
}

func (s *EntcollectStore) Each(fn func(key string, decode func(any) error) (bool, error)) error {
	err := s.tx.ForEach(s.bucket, func(k, v []byte) error {
		cont, err := fn(string(k), func(dst any) error {
			return json.Unmarshal(v, dst)
		})
		if err != nil {
			return err
		}
		if !cont {
			return errStopIteration
		}
		return nil
	})
	if errors.Is(err, errStopIteration) {
		return nil
	}
	if errors.Is(err, ErrBucketNotFound) {
		return nil
	}
	return err
}

var errStopIteration = errors.New("stop iteration")
