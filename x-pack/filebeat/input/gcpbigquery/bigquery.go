// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package gcpbigquery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/iterator"

	"github.com/elastic/elastic-agent-libs/logp"
)

// internal interfaces for testing
type client interface {
	query(query string, params map[string]interface{}) query
}

type query interface {
	read(ctx context.Context) (rowIterator, error)
}

type rowIterator interface {
	next(val *[]bigquery.Value) error
	schema() bigquery.Schema
}

// adapter for the real BigQuery client
type realClient struct {
	*bigquery.Client
}

func (r *realClient) query(queryString string, params map[string]interface{}) query {
	q := r.Query(queryString)
	for k, v := range params {
		q.Parameters = append(q.Parameters, bigquery.QueryParameter{
			Name:  k,
			Value: v,
		})
	}
	return &realQuery{q}

}

type realQuery struct {
	*bigquery.Query
}

func (r *realQuery) read(ctx context.Context) (rowIterator, error) {
	it, err := r.Read(ctx)
	if err != nil {
		return nil, err
	}
	return &realRowIterator{it}, nil
}

type realRowIterator struct {
	*bigquery.RowIterator
}

func (r *realRowIterator) next(row *[]bigquery.Value) error {
	return r.Next(row)
}

func (r *realRowIterator) schema() bigquery.Schema {
	return r.Schema
}

func runQuery(ctx context.Context, logger *logp.Logger, client *bigquery.Client, queryString string, params map[string]interface{}, publish func(bigquery.Schema, []bigquery.Value)) error {
	return runQueryInternal(ctx, logger, &realClient{client}, queryString, params, publish)
}

func runQueryInternal(ctx context.Context, logger *logp.Logger, client client, queryString string, params map[string]interface{}, publish func(bigquery.Schema, []bigquery.Value)) error {
	logger.Debugf("executing query: '%s' with params: %v", queryString, params)

	query := client.query(queryString, params)
	it, err := query.read(ctx)
	if err != nil {
		return err
	}

	for {
		var row []bigquery.Value
		err := it.next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			logger.Errorf("failed to iterate bigquery result: %v", err)
			return err
		}

		publish(it.schema(), row)
	}

	return nil
}

func expandJSON(field *bigquery.FieldSchema, value bigquery.Value) (interface{}, bool, error) {
	expanded := false
	if value == nil {
		return nil, expanded, nil
	}

	stringVal, ok := (value).(string)

	if !ok {
		return value, expanded, nil
	}

	if field.Type != bigquery.JSONFieldType {
		return value, expanded, nil
	}

	// For JSON fields, parse the string into a map or slice.
	var jsonData interface{}
	if err := json.Unmarshal([]byte(stringVal), &jsonData); err != nil {
		return nil, expanded, err
	}

	expanded = true
	return jsonData, expanded, nil
}

func getTimestamp(field *bigquery.FieldSchema, value bigquery.Value) (time.Time, error) {
	var timestamp time.Time

	if field.Type != bigquery.TimestampFieldType {
		return timestamp, fmt.Errorf("timestamp_field is not of type TIMESTAMP")
	}

	timestamp, ok := value.(time.Time)
	if !ok {
		return timestamp, fmt.Errorf("timestamp_field is not time.Time")
	}

	return timestamp, nil
}

// cursorState holds the last cursor value
type cursorState struct {
	FieldType string
	StringVal string
}

func (c *cursorState) set(field *bigquery.FieldSchema, value bigquery.Value) error {
	var stringVal string
	var err error

	switch field.Type {
	case bigquery.StringFieldType:
		stringVal, err = serialize(value, field.Type, func(v string) string { return v })
	case bigquery.IntegerFieldType:
		stringVal, err = serialize(value, field.Type, func(v int64) string { return strconv.FormatInt(v, 10) })
	case bigquery.FloatFieldType:
		stringVal, err = serialize(value, field.Type, func(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) })
	case bigquery.BytesFieldType:
		stringVal, err = serialize(value, field.Type, func(v []byte) string { return base64.StdEncoding.EncodeToString(v) })
	case bigquery.TimestampFieldType:
		stringVal, err = serialize(value, field.Type, func(v time.Time) string { return v.UTC().Format(time.RFC3339Nano) })
	case bigquery.DateFieldType:
		stringVal, err = serialize(value, field.Type, func(v civil.Date) string { return v.String() })
	case bigquery.TimeFieldType:
		stringVal, err = serialize(value, field.Type, func(v civil.Time) string { return v.String() })
	case bigquery.DateTimeFieldType:
		stringVal, err = serialize(value, field.Type, func(v civil.DateTime) string { return v.String() })
	case bigquery.NumericFieldType, bigquery.BigNumericFieldType:
		stringVal, err = serialize(value, field.Type, func(v *big.Rat) string { return v.String() })
	default:
		err = fmt.Errorf("unsupported field type: %s", field.Type)
	}

	if err != nil {
		return fmt.Errorf("cannot serialize cursor value: %w", err)
	}

	c.StringVal = stringVal
	c.FieldType = string(field.Type)
	return nil
}

func (c *cursorState) get() (bigquery.Value, error) {
	var val bigquery.Value
	var err error

	switch bigquery.FieldType(c.FieldType) {
	case bigquery.StringFieldType:
		val = c.StringVal
	case bigquery.IntegerFieldType:
		val, err = strconv.ParseInt(c.StringVal, 10, 64)
	case bigquery.FloatFieldType:
		val, err = strconv.ParseFloat(c.StringVal, 64)
	case bigquery.BytesFieldType:
		val, err = base64.StdEncoding.DecodeString(c.StringVal)
	case bigquery.TimestampFieldType:
		val, err = time.Parse(time.RFC3339Nano, c.StringVal)
	case bigquery.DateFieldType:
		val, err = civil.ParseDate(c.StringVal)
	case bigquery.TimeFieldType:
		val, err = civil.ParseTime(c.StringVal)
	case bigquery.DateTimeFieldType:
		val, err = civil.ParseDateTime(c.StringVal)
	case bigquery.NumericFieldType, bigquery.BigNumericFieldType:
		v := new(big.Rat)
		if _, ok := v.SetString(c.StringVal); ok {
			val = v
		} else {
			err = fmt.Errorf("invalid big.Rat")
		}
	default:
		err = fmt.Errorf("unsupported field type")
	}

	if err != nil {
		return nil, fmt.Errorf("cannot deserialize cursor value '%s' as %s: %w", c.StringVal, c.FieldType, err)
	}

	return val, nil
}

func serialize[T any](value bigquery.Value, t bigquery.FieldType, converter func(T) string) (string, error) {
	if v, ok := value.(T); ok {
		return converter(v), nil
	}

	return "", fmt.Errorf("unexpected type for %s field, got %T (%v)", t, value, value)
}
