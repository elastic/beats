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

package validate

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// Document takes a document from Elasticsearch in JSON format
// and the contents of a Beat's fields.yml and validated the document's
// fields against it.
func Document(docJSON []byte, fieldsYAML []byte) error {
	var ifDocument interface{}
	if err := json.Unmarshal(docJSON, &ifDocument); err != nil {
		return errors.Wrap(err, "decoding JSON document")
	}

	document, ok := ifDocument.(map[string]interface{})
	if !ok {
		return errors.Errorf("document must be a dictionary of string keys, but %T", ifDocument)
	}

	fields, err := NewMapping(fieldsYAML)
	if err != nil {
		return err
	}
	return fields.Validate(document)
}
