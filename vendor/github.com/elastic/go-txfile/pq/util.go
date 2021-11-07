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

package pq

import "github.com/elastic/go-txfile"

func getPage(tx *txfile.Tx, id txfile.PageID) ([]byte, error) {
	page, err := tx.Page(id)
	if err != nil {
		return nil, err
	}

	return page.Bytes()
}

func withPage(tx *txfile.Tx, id txfile.PageID, fn func([]byte)) error {
	page, err := getPage(tx, id)
	if err != nil {
		return err
	}

	fn(page)
	return nil
}

func idLess(a, b uint64) bool {
	return int64(a-b) < 0
}

func idLessEq(a, b uint64) bool {
	return a == b || idLess(a, b)
}
