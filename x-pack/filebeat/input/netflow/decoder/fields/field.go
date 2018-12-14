// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fields

import "fmt"

var Fields = FieldDict{}

type Key struct {
	EnterpriseID uint32
	FieldID      uint16
}

type Field struct {
	Name    string
	Decoder Decoder
}

type FieldDict map[Key]*Field

func RegisterFields(dict FieldDict) error {
	for key, value := range dict {
		if _, found := Fields[key]; found {
			return fmt.Errorf("field %+v is duplicated", key)
		}
		Fields[key] = value
	}
	return nil
}
