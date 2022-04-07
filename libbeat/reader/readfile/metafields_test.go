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

package readfile

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/reader"
)

func TestMetaFieldsOffset(t *testing.T) {
	messages := []reader.Message{
		reader.Message{
			Content: []byte("my line"),
			Bytes:   7,
			Fields:  common.MapStr{},
		},
		reader.Message{
			Content: []byte("my line again"),
			Bytes:   13,
			Fields:  common.MapStr{},
		},
		reader.Message{
			Content: []byte(""),
			Bytes:   10,
			Fields:  common.MapStr{},
		},
	}

	path := "test/path"
	offset := int64(0)
	in := &FileMetaReader{msgReader(messages), path, offset}
	for {
		msg, err := in.Next()
		if err == io.EOF {
			break
		}

		expectedFields := common.MapStr{}
		if len(msg.Content) != 0 {
			expectedFields = common.MapStr{
				"log": common.MapStr{
					"file": common.MapStr{
						"path": path,
					},
					"offset": offset,
				},
			}
		}
		offset += int64(msg.Bytes)

		require.Equal(t, expectedFields, msg.Fields)
		require.Equal(t, offset, in.offset)
	}
}

func msgReader(m []reader.Message) reader.Reader {
	return &messageReader{
		messages: m,
	}
}

type messageReader struct {
	messages []reader.Message
	i        int
}

func (r *messageReader) Next() (reader.Message, error) {
	if r.i == len(r.messages) {
		return reader.Message{}, io.EOF
	}
	msg := r.messages[r.i]
	r.i++
	return msg, nil
}

func (r *messageReader) Close() error {
	return nil
}
