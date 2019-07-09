package oracle

import (
	"database/sql"
	"errors"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// checkNullString avoid setting an invalid empty string value on Metricbeat event
func CheckNullSqlValue(logger *logp.Logger, output map[string]common.MapStr, parentTargetFieldName, targetFieldName string, v SqlValue) {
	if v.isValid() {
		if _, ok := output[parentTargetFieldName]; ok {
			if _, err := output[parentTargetFieldName].Put(targetFieldName, v.Value()); err != nil {
				logger.Debug(errors.New("error trying to set value on common.Mapstr"))
			}
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
