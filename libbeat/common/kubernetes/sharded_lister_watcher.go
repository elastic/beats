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

package kubernetes

import (
	"github.com/cespare/xxhash/v2"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type shardedListWatch struct {
	lw       cache.ListerWatcher
	count    uint64
	instance uint64
}

func NewShardedListWatch(instance int, count int, lw cache.ListerWatcher) cache.ListerWatcher {
	if count == 0 {
		return lw
	}

	return &shardedListWatch{
		lw:       lw,
		count:    uint64(count),
		instance: uint64(instance),
	}
}

func (s *shardedListWatch) List(options metav1.ListOptions) (runtime.Object, error) {
	list, err := s.lw.List(options)
	if err != nil {
		return nil, err
	}

	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, err
	}

	res := &metav1.List{
		Items: make([]runtime.RawExtension, 0, len(items)),
	}

	for _, item := range items {
		a, err := meta.Accessor(item)
		if err != nil {
			return nil, err
		}

		if s.filter(a) {
			res.Items = append(res.Items, runtime.RawExtension{Object: item})
		}
	}

	return res, nil
}

func (s *shardedListWatch) Watch(options metav1.ListOptions) (watch.Interface, error) {
	w, err := s.lw.Watch(options)
	if err != nil {
		return nil, err
	}

	return watch.Filter(w, func(in watch.Event) (out watch.Event, keep bool) {
		a, err := meta.Accessor(in.Object)
		if err != nil {
			return in, true
		}

		return in, s.filter(a)
	}), nil
}

func (s *shardedListWatch) filter(o metav1.Object) bool {
	h := xxhash.New()
	h.Write([]byte(o.GetUID()))
	return (h.Sum64() % s.count) == s.instance
}
