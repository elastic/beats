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

package decode_csv_fields

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

type decodeCSVFields struct {
	csvConfig
	fields    map[string]string
	separator rune
	headers   map[string]csvHeader
}

type csvConfig struct {
	Fields           common.MapStr `config:"fields"`
	IgnoreMissing    bool          `config:"ignore_missing"`
	TrimLeadingSpace bool          `config:"trim_leading_space"`
	OverwriteKeys    bool          `config:"overwrite_keys"`
	FailOnError      bool          `config:"fail_on_error"`
	Separator        string        `config:"separator"`
	Headers          common.MapStr `config:"headers`
}

type csvHeader struct {
	custom bool   `config:"in_file"`
	header string `config:"string"`
	offset int    `config:"offset"`
	file   `config:"file"`
}

type file struct {
	path string `config:"path"`
}

var (
	defaultCSVConfig = csvConfig{
		Separator:   ",",
		FailOnError: true,
	}

	errFieldAlreadySet = errors.New("field already has a value")
)

func init() {
	processors.RegisterPlugin("decode_csv_fields",
		checks.ConfigChecked(NewDecodeCSVField,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "ignore_missing", "overwrite_keys", "separator", "trim_leading_space", "overwrite_keys", "fail_on_error", "when",
				"headers", "file", "string", "offset", "path", "in_file")))

	jsprocessor.RegisterPlugin("DecodeCSVField", NewDecodeCSVField)
}

// NewDecodeCSVField construct a new decode_csv_field processor.
func NewDecodeCSVField(c *common.Config) (processors.Processor, error) {
	config := defaultCSVConfig

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the decode_csv_field configuration: %s", err)
	}
	if len(config.Fields) == 0 {
		return nil, errors.New("no fields to decode configured")
	}
	f := &decodeCSVFields{csvConfig: config}
	// Set separator as rune
	switch runes := []rune(config.Separator); len(runes) {
	case 0:
		break
	case 1:
		f.separator = runes[0]
	default:
		return nil, errors.Errorf("separator must be a single character, got %d in string '%s'", len(runes), config.Separator)
	}
	// Set fields as string -> string
	f.fields = make(map[string]string, len(config.Fields))
	for src, dstIf := range config.Fields.Flatten() {
		dst, ok := dstIf.(string)
		if !ok {
			return nil, errors.Errorf("bad destination mapping for %s: destination field must be string, not %T (got %v)", src, dstIf, dstIf)
		}
		f.fields[src] = dst
	}
	// Set headers as string -> csvHeader
	f.headers = make(map[string]csvHeader, len(config.Headers))
	for src, dstIf := range config.Headers {
		var dst map[string]interface{}
		var isString int = 0
		var isFile int = 0
		var isCustom int = 0
		var isOffset bool = false
		jsonString, _ := json.Marshal(dstIf)
		json.Unmarshal([]byte(jsonString), &dst)
		var toHeader csvHeader
		if val1, found := dst["string"]; found {
			val2, ok := val1.(string)
			if !ok {
				return nil, errors.Errorf("bad destination mapping for \"string\": destination field must be string, not %T (got %v)", val1, val1)
			}
			toHeader.header = val2
			isString = 1
		}
		if val1, found := dst["offset"]; found {
			_, err := val1.(float64)
			val2, ok := strconv.Atoi(fmt.Sprintf("%v", val1))
			if ok != nil || !err {
				return nil, errors.Errorf("bad destination mapping for \"offset\": destination field must be int, got %v", val1)
			}
			if val2 <= 0 {
				return nil, errors.Errorf("\"offset\" must bigger than 0 (got %v)", val2)
			}
			toHeader.offset = val2
			isOffset = true
		}
		if val1, found := dst["in_file"]; found {
			val2, ok := val1.(bool)
			if !ok {
				return nil, errors.Errorf("bad destination mapping for \"in_file\": destination field must be bool, not %T (got %v)", val1, val1)
			}
			if !val2 {
				return nil, errors.Errorf("\"in_file\" must be set to true")
			}
			toHeader.custom = true
			isCustom = 1
		}
		if val1, found := dst["file"]; found {
			jsonString, _ = json.Marshal(val1)
			json.Unmarshal([]byte(jsonString), &dst)
			if val2, found := dst["path"]; found {
				val3, ok := val2.(string)
				if !ok {
					return nil, errors.Errorf("bad destination mapping for \"file.path\": destination field must be string, not %T (got %v)", val2, val2)
				}
				toHeader.file.path = val3
				isFile = 1
			}
		}
		//can't use all configuration
		if isString+isFile+isCustom != 1 {
			return nil, errors.Errorf("choose one configuration : \"string\" or \"file\" or \"in_file\"")
		}
		//can't use string and offset
		if isString == 1 && isOffset {
			return nil, errors.Errorf("you cannot use \"string\" with \"offset\"")
		}
		f.headers[src] = toHeader
	}
	return f, nil
}

// Run applies the decode_csv_field processor to an event.
func (f *decodeCSVFields) Run(event *beat.Event) (*beat.Event, error) {
	saved := *event
	if f.FailOnError {
		saved.Fields = event.Fields.Clone()
		saved.Meta = event.Meta.Clone()
	}
	for src, dest := range f.fields {
		if err := f.decodeCSVField(src, dest, event); err != nil && f.FailOnError {
			return &saved, err
		}
	}
	return event, nil
}

func (f *decodeCSVFields) decodeCSVField(src, dest string, event *beat.Event) error {
	data, err := event.GetValue(src)
	if err != nil {
		if f.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return nil
		}
		return errors.Wrapf(err, "could not fetch value for field %s", src)
	}

	text, ok := data.(string)
	if !ok {
		return errors.Errorf("field %s is not of string type", src)
	}

	reader := csv.NewReader(strings.NewReader(text))
	reader.Comma = f.separator
	reader.TrimLeadingSpace = f.TrimLeadingSpace
	// LazyQuotes makes the parser more tolerant to bad string formatting.
	reader.LazyQuotes = true

	record, err := reader.Read()
	if err != nil {
		return errors.Wrapf(err, "error decoding CSV from field %s", src)
	}

	if src != dest && !f.OverwriteKeys {
		if _, err = event.GetValue(dest); err == nil {
			return errors.Errorf("target field %s already has a value. Set the overwrite_keys flag or drop/rename the field first", dest)
		}
	}
	if _, exist := f.Headers[src]; exist == false {
		if _, err = event.PutValue(dest, record); err != nil {
			return errors.Wrapf(err, "failed setting field %s", dest)
		}
		return nil
	}

	firstLine := ""
	head, _ := f.headers[src]
	if len(head.header) > 0 {
		//header in .yml
		firstLine = head.header
	} else {
		strPath := ""
		if len(head.file.path) > 0 {
			//header in file conf
			strPath = head.file.path
		} else {
			//header in current file
			//get path file and open file
			path, err := event.GetValue("log.file.path")
			if err != nil {
				return errors.Errorf("could not fetch value for field log.file.path")
			}
			strPath = fmt.Sprintf("%v", path)
		}
		//if doesn't give offset, set to default 1
		if head.offset == 0 {
			head.offset = 1
		}
		file, err := os.Open(strPath)
		if err != nil {
			return errors.Errorf("could not open file : %v", strPath)
		}
		defer file.Close()

		//read header in file
		scanner := bufio.NewScanner(file)
		for scanner.Scan() && head.offset > 0 {
			firstLine = scanner.Text()
			head.offset--
			if err := scanner.Err(); err != nil {
				return errors.Errorf("error from scanner.Text() in read file")
			}
		}
		if head.offset != 0 {
			return errors.Errorf("error: offset too large")
		}
	}
	if text == firstLine {
		return nil
	}
	//get header record
	reader = csv.NewReader(strings.NewReader(firstLine))
	reader.Comma = f.separator
	reader.TrimLeadingSpace = f.TrimLeadingSpace
	reader.LazyQuotes = true
	headcsv, err := reader.Read()
	if err != nil {
		return errors.Errorf("error decoding header")
	}
	//check length header and record
	if len(headcsv) != len(record) {
		return errors.Errorf("number of header and line are different, given %v headers, needed %v", len(headcsv), len(record))
	}
	//create map object
	mymap := make(map[string]string)
	for i := 0; i < len(headcsv); i++ {
		if headcsv[i] == "" {
			headcsv[i] = fmt.Sprintf("%v", i)
		}
		if record[i] == "" {
			record[i] = fmt.Sprintf("%v", i)
		}
		mymap[headcsv[i]] = record[i]
	}

	//put result in fields dest
	if _, err = event.PutValue(dest, mymap); err != nil {
		return errors.Wrapf(err, "failed setting field %s", dest)
	}
	return nil
}

// String returns a string representation of this processor.
func (f decodeCSVFields) String() string {
	json, _ := json.Marshal(f.csvConfig)
	return "decode_csv_field=" + string(json)
}
