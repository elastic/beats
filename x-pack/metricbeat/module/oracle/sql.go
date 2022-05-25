// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package oracle

import (
	"database/sql"
	"errors"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// SetSqlValueWithParentKey avoid setting an invalid empty value value on Metricbeat event
func SetSqlValueWithParentKey(logger *logp.Logger, output map[string]mapstr.M, parentTargetFieldName, targetFieldName string, v SqlValue) {
	ms, ok := output[parentTargetFieldName]
	if !ok {
		logger.Debug(errors.New("no parent key found"))
		return
	}

	SetSqlValue(logger, ms, targetFieldName, v)
}

//SetSqlValue avoid setting an invalid empty value value on Metricbeat event
func SetSqlValue(logger *logp.Logger, output mapstr.M, targetFieldName string, v SqlValue) {
	if v.isValid() {
		if _, err := output.Put(targetFieldName, v.Value()); err != nil {
			logger.Debug(errors.New("error trying to set value on common.Mapstr"))
		}
	} else {
		logger.Debug(errors.New("invalid value returned from database (null)"))
	}

	return
}

func NewSqlWrapper(fieldName string, value SqlValue) *SqlValueWrapper {
	return &SqlValueWrapper{
		Field:    fieldName,
		SqlValue: value,
	}
}

type SqlValueWrapper struct {
	Field    string
	SqlValue SqlValue
}

type SqlValue interface {
	isValid() bool
	Value() interface{}
}

type Float64Value struct {
	sql.NullFloat64
}

func (i *Float64Value) isValid() bool {
	return i.Valid
}

func (i *Float64Value) Value() interface{} {
	return i.Float64
}

type Int64Value struct {
	sql.NullInt64
}

func (i *Int64Value) isValid() bool {
	return i.Valid
}

func (i *Int64Value) Value() interface{} {
	return i.Int64
}

type StringValue struct {
	sql.NullString
}

func (s *StringValue) isValid() bool {
	return s.Valid
}

func (s *StringValue) Value() interface{} {
	return s.String
}
