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

//go:build windows
// +build windows

package wineventlog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBookmark(t *testing.T) {
	log := openLog(t, security4752File)
	defer log.Close()

	evtHandle := mustNextHandle(t, log)
	defer evtHandle.Close()

	t.Run("NewBookmarkFromEvent", func(t *testing.T) {
		bookmark, err := NewBookmarkFromEvent(evtHandle)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			assert.NoError(t, bookmark.Close())
		}()

		xml, err := bookmark.XML()
		if err != nil {
			t.Fatal(err)
		}

		assert.Contains(t, xml, "<BookmarkList", "</BookmarkList>")
	})

	t.Run("NewBookmarkFromXML", func(t *testing.T) {
		const savedBookmarkXML = `
<BookmarkList>
  <Bookmark Channel='Dummy' RecordId='1' IsCurrent='true'/>
</BookmarkList>`

		bookmark, err := NewBookmarkFromXML(savedBookmarkXML)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			assert.NoError(t, bookmark.Close())
		}()

		xml, err := bookmark.XML()
		if err != nil {
			t.Fatal(err)
		}

		// Ignore whitespace differences.
		normalizer := strings.NewReplacer(" ", "", "\r\n", "", "\n", "")
		assert.Equal(t, normalizer.Replace(savedBookmarkXML), normalizer.Replace(xml))
	})

	t.Run("NewBookmarkFromEvent_invalid", func(t *testing.T) {
		bookmark, err := NewBookmarkFromEvent(NilHandle)
		assert.Error(t, err)
		assert.Zero(t, bookmark)
	})

	t.Run("NewBookmarkFromXML_invalid", func(t *testing.T) {
		bookmark, err := NewBookmarkFromXML("{Not XML}")
		assert.Error(t, err)
		assert.Zero(t, bookmark)
	})
}
