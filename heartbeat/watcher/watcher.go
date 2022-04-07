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

package watcher

import (
	"hash/fnv"
	"io/ioutil"
	"os"
	"time"

	"github.com/elastic/beats/v8/libbeat/logp"
)

type Watch interface {
	Stop()
}

type filePoller struct {
	done chan struct{}
}

type fileChangeTester struct {
	path string
	sz   int
	hash uint64
	stat os.FileInfo
}

func NewFilePoller(
	path string,
	poll time.Duration,
	do func([]byte),
) (Watch, error) {
	fw := &filePoller{
		done: make(chan struct{}),
	}

	tester := &fileChangeTester{path: path, sz: -1}
	if content, changed := tester.check(); changed {
		do(content)
	}

	go func() {
		ticker := time.NewTicker(poll)
		defer ticker.Stop()

		for {
			if content, changed := tester.check(); changed {
				do(content)
			}

			select {
			case <-fw.done:
				return
			case <-ticker.C:
			}
		}
	}()

	return fw, nil
}

func (f *filePoller) Stop() {
	close(f.done)
}

func (w *fileChangeTester) check() ([]byte, bool) {
	f, err := os.Open(w.path)
	if err != nil {
		logp.Info("Failed to load file: %v", err)
		return nil, false
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		logp.Info("Reading file '%v' stat failed with: %v", w.path, err)
		return nil, false
	}

	if w.stat != nil {
		if stat.Size() == w.stat.Size() && !stat.ModTime().After(w.stat.ModTime()) {
			return nil, false
		}
	}

	content, err := ioutil.ReadAll(f)
	if err != nil {
		logp.Info("Reading file '%v' failed with: %v", w.path, err)
		return nil, false
	}

	var hash uint64
	if len(content) != 0 || w.sz == 0 {
		hasher := fnv.New64a()
		hasher.Write(content)
		hash = hasher.Sum64()
		if w.hash == hash {
			return nil, false
		}
	}

	w.hash = hash
	w.stat = stat
	w.sz = len(content)
	return content, true
}
