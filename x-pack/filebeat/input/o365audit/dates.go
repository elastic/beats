// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"sort"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

const (
	// Date format used by audit objects.
	apiDateFormat = "2006-01-02T15:04:05"
	timeDay       = time.Hour * 24
)

var (
	errTypeCastFailed = errors.New("key is not expected type")
)

// Date formats used in the JSON objects returned by the API.
// This is just a safeguard in case the date format used by the API is
// updated to include sub-second resolution or timezone information.
var apiDateFormats = dateFormats{
	apiDateFormat,
	apiDateFormat + "Z",
	time.RFC3339Nano,
	time.RFC3339,
}

// Date formats used by HTTP/1.1 servers.
var httpDateFormats = dateFormats{
	time.RFC1123,
	time.RFC850,
	time.ANSIC,
	time.RFC1123Z,
}

// A helper to parse dates using different formats.
type dateFormats []string

// Parse will try to parse the given string-formatted date in the formats
// specified in the dateFormats until one succeeds.
func (d dateFormats) Parse(str string) (t time.Time, err error) {
	for _, fmt := range d {
		if t, err = time.Parse(fmt, str); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Now().UTC(), fmt.Errorf("unable to parse date '%s' with formats %v", str, d)
}

// Get a key from a map and cast it to string.
func getString(m common.MapStr, key string) (string, error) {
	iValue, err := m.GetValue(key)
	if err != nil {
		return "", err
	}
	str, ok := iValue.(string)
	if !ok {
		return "", errTypeCastFailed
	}
	return str, nil
}

// Parse a date from the given map key.
func getDateKey(m common.MapStr, key string, formats dateFormats) (t time.Time, err error) {
	str, err := getString(m, key)
	if err != nil {
		return t, err
	}
	return formats.Parse(str)
}

// Sort a slice of maps by one of its keys parsed as a date in the given format(s).
func sortMapSliceByDate(s []common.MapStr, dateKey string, formats dateFormats) error {
	var errs multierror.Errors
	sort.Slice(s, func(i, j int) bool {
		di, e1 := getDateKey(s[i], dateKey, formats)
		dj, e2 := getDateKey(s[j], dateKey, formats)
		if e1 != nil {
			errs = append(errs, e1)
		}
		if e2 != nil {
			errs = append(errs, e2)
		}
		return di.Before(dj)
	})
	return errors.Wrapf(errs.Err(), "failed sorting by date key:%s", dateKey)
}

func inRange(d, maxLimit time.Duration) bool {
	if maxLimit < 0 {
		maxLimit = -maxLimit
	}
	if d < 0 {
		d = -d
	}
	return d < maxLimit
}
