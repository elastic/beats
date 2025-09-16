package gcpbigquery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/iterator"

	"github.com/elastic/elastic-agent-libs/logp"
)

// internal interfaces for testing
type client interface {
	query(string) query
}

type query interface {
	read(context.Context) (rowIterator, error)
}

type rowIterator interface {
	next(*[]bigquery.Value) error
	schema() bigquery.Schema
}

// adapter for the real BigQuery client
type realClient struct {
	*bigquery.Client
}

func (r *realClient) query(queryString string) query {
	return &realQuery{r.Query(queryString)}
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

func runQuery(ctx context.Context, logger *logp.Logger, client *bigquery.Client, queryString string, publish func(bigquery.Schema, []bigquery.Value)) error {
	return runQueryInternal(ctx, logger, &realClient{client}, queryString, publish)
}

func runQueryInternal(ctx context.Context, logger *logp.Logger, client client, queryString string, publish func(bigquery.Schema, []bigquery.Value)) error {
	logger.Debugf("executing query: %s", queryString)

	query := client.query(queryString)
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

func expandJSON(field *bigquery.FieldSchema, value bigquery.Value) (interface{}, error) {
	if value == nil {
		return nil, nil
	}

	stringVal, ok := (value).(string)

	if !ok {
		return value, nil
	}

	if field.Type != bigquery.JSONFieldType {
		return value, nil
	}

	// For JSON fields, parse the string into a map or slice.
	var jsonData interface{}
	if err := json.Unmarshal([]byte(stringVal), &jsonData); err != nil {
		return nil, err
	}

	return jsonData, nil
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

// cursorState holds the stringified last cursor value
type cursorState struct {
	WhereVal string
}

func (c *cursorState) set(field *bigquery.FieldSchema, value bigquery.Value) error {
	errorMsg := "expected %s value for %s field, got %T"

	switch field.Type {
	case bigquery.StringFieldType:
		v, ok := value.(string)
		if !ok {
			return fmt.Errorf(errorMsg, "string", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("\"%s\"", v)
	case bigquery.IntegerFieldType:
		v, ok := value.(int64)
		if !ok {
			return fmt.Errorf(errorMsg, "int64", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("%d", v)
	case bigquery.FloatFieldType:
		v, ok := value.(float64)
		if !ok {
			return fmt.Errorf(errorMsg, "float64", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("%f", v)
	case bigquery.BytesFieldType:
		v, ok := value.([]byte)
		if !ok {
			return fmt.Errorf(errorMsg, "[]byte", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("B\"%s\"", v)
	case bigquery.TimestampFieldType:
		v, ok := value.(time.Time)
		if !ok {
			return fmt.Errorf(errorMsg, "time.Time", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("TIMESTAMP '%s'", v.UTC().Format("2006-01-02T15:04:05.999999Z"))
	case bigquery.DateFieldType:
		v, ok := value.(civil.Date)
		if !ok {
			return fmt.Errorf(errorMsg, "civil.Date", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("DATE '%s'", v.String())
	case bigquery.TimeFieldType:
		v, ok := value.(civil.Time)
		if !ok {
			return fmt.Errorf(errorMsg, "civil.Time", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("TIME '%s'", bigquery.CivilTimeString(v))
	case bigquery.DateTimeFieldType:
		v, ok := value.(civil.DateTime)
		if !ok {
			return fmt.Errorf(errorMsg, "civil.DateTime", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("DATETIME '%s'", bigquery.CivilDateTimeString(v))
	case bigquery.NumericFieldType:
		v, ok := value.(*big.Rat)
		if !ok {
			return fmt.Errorf(errorMsg, "*big.Rat", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("NUMERIC '%s'", bigquery.NumericString(v))
	case bigquery.BigNumericFieldType:
		v, ok := value.(*big.Rat)
		if !ok {
			return fmt.Errorf(errorMsg, "*big.Rat", field.Type, value)
		}
		c.WhereVal = fmt.Sprintf("BIGNUMERIC '%s'", bigquery.BigNumericString(v))
	default:
		return fmt.Errorf("unsupported cursor field type: %s", field.Type)
	}

	return nil
}
