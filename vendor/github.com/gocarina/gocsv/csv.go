// Copyright 2014 Jonathan Picques. All rights reserved.
// Use of this source code is governed by a MIT license
// The license can be found in the LICENSE file.

// The GoCSV package aims to provide easy CSV serialization and deserialization to the golang programming language

package gocsv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// FailIfUnmatchedStructTags indicates whether it is considered an error when there is an unmatched
// struct tag.
var FailIfUnmatchedStructTags = false

// FailIfDoubleHeaderNames indicates whether it is considered an error when a header name is repeated
// in the csv header.
var FailIfDoubleHeaderNames = false

// ShouldAlignDuplicateHeadersWithStructFieldOrder indicates whether we should align duplicate CSV
// headers per their alignment in the struct definition.
var ShouldAlignDuplicateHeadersWithStructFieldOrder = false

// TagSeparator defines seperator string for multiple csv tags in struct fields
var TagSeparator = ","

// --------------------------------------------------------------------------
// CSVWriter used to format CSV

var selfCSVWriter = DefaultCSVWriter

// DefaultCSVWriter is the default CSV writer used to format CSV (cf. csv.NewWriter)
func DefaultCSVWriter(out io.Writer) *csv.Writer {
	writer := csv.NewWriter(out)

	// As only one rune can be defined as a CSV separator, we are going to trim
	// the custom tag separator and use the first rune.
	if runes := []rune(strings.TrimSpace(TagSeparator)); len(runes) > 0 {
		writer.Comma = runes[0]
	}

	return writer
}

// SetCSVWriter sets the CSV writer used to format CSV.
func SetCSVWriter(csvWriter func(io.Writer) *csv.Writer) {
	selfCSVWriter = csvWriter
}

func getCSVWriter(out io.Writer) *csv.Writer {
	return selfCSVWriter(out)
}

// --------------------------------------------------------------------------
// CSVReader used to parse CSV

var selfCSVReader = DefaultCSVReader

// DefaultCSVReader is the default CSV reader used to parse CSV (cf. csv.NewReader)
func DefaultCSVReader(in io.Reader) *csv.Reader {
	return csv.NewReader(in)
}

// LazyCSVReader returns a lazy CSV reader, with LazyQuotes and TrimLeadingSpace.
func LazyCSVReader(in io.Reader) *csv.Reader {
	csvReader := csv.NewReader(in)
	csvReader.LazyQuotes = true
	csvReader.TrimLeadingSpace = true
	return csvReader
}

// SetCSVReader sets the CSV reader used to parse CSV.
func SetCSVReader(csvReader func(io.Reader) *csv.Reader) {
	selfCSVReader = csvReader
}

func getCSVReader(in io.Reader) *csv.Reader {
	return selfCSVReader(in)
}

// --------------------------------------------------------------------------
// Marshal functions

// MarshalFile saves the interface as CSV in the file.
func MarshalFile(in interface{}, file *os.File) (err error) {
	return Marshal(in, file)
}

// MarshalString returns the CSV string from the interface.
func MarshalString(in interface{}) (out string, err error) {
	bufferString := bytes.NewBufferString(out)
	if err := Marshal(in, bufferString); err != nil {
		return "", err
	}
	return bufferString.String(), nil
}

// MarshalBytes returns the CSV bytes from the interface.
func MarshalBytes(in interface{}) (out []byte, err error) {
	bufferString := bytes.NewBuffer(out)
	if err := Marshal(in, bufferString); err != nil {
		return nil, err
	}
	return bufferString.Bytes(), nil
}

// Marshal returns the CSV in writer from the interface.
func Marshal(in interface{}, out io.Writer) (err error) {
	writer := getCSVWriter(out)
	return writeTo(writer, in, false)
}

// Marshal returns the CSV in writer from the interface.
func MarshalWithoutHeaders(in interface{}, out io.Writer) (err error) {
	writer := getCSVWriter(out)
	return writeTo(writer, in, true)
}

// MarshalChan returns the CSV read from the channel.
func MarshalChan(c <-chan interface{}, out *csv.Writer) error {
	return writeFromChan(out, c)
}

// MarshalCSV returns the CSV in writer from the interface.
func MarshalCSV(in interface{}, out *csv.Writer) (err error) {
	return writeTo(out, in, false)
}

// --------------------------------------------------------------------------
// Unmarshal functions

// UnmarshalFile parses the CSV from the file in the interface.
func UnmarshalFile(in *os.File, out interface{}) (err error) {
	return Unmarshal(in, out)
}

// UnmarshalString parses the CSV from the string in the interface.
func UnmarshalString(in string, out interface{}) (err error) {
	return Unmarshal(strings.NewReader(in), out)
}

// UnmarshalBytes parses the CSV from the bytes in the interface.
func UnmarshalBytes(in []byte, out interface{}) (err error) {
	return Unmarshal(bytes.NewReader(in), out)
}

// Unmarshal parses the CSV from the reader in the interface.
func Unmarshal(in io.Reader, out interface{}) (err error) {
	return readTo(newDecoder(in), out)
}

// UnmarshalCSV parses the CSV from the reader in the interface.
func UnmarshalCSV(in CSVReader, out interface{}) error {
	return readTo(csvDecoder{in}, out)
}

// UnmarshalToChan parses the CSV from the reader and send each value in the chan c.
// The channel must have a concrete type.
func UnmarshalToChan(in io.Reader, c interface{}) error {
	if c == nil {
		return fmt.Errorf("goscv: channel is %v", c)
	}
	return readEach(newDecoder(in), c)
}

// UnmarshalStringToChan parses the CSV from the string and send each value in the chan c.
// The channel must have a concrete type.
func UnmarshalStringToChan(in string, c interface{}) error {
	return UnmarshalToChan(strings.NewReader(in), c)
}

// UnmarshalBytesToChan parses the CSV from the bytes and send each value in the chan c.
// The channel must have a concrete type.
func UnmarshalBytesToChan(in []byte, c interface{}) error {
	return UnmarshalToChan(bytes.NewReader(in), c)
}

// UnmarshalToCallback parses the CSV from the reader and send each value to the given func f.
// The func must look like func(Struct).
func UnmarshalToCallback(in io.Reader, f interface{}) (err error) {
	valueFunc := reflect.ValueOf(f)
	t := reflect.TypeOf(f)
	if t.NumIn() != 1 {
		return fmt.Errorf("the given function must have exactly one parameter")
	}
	c := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, t.In(0)), 0)
	go func() {
		err = UnmarshalToChan(in, c.Interface())
	}()
	for {
		if err != nil {
			return err
		}
		v, notClosed := c.Recv()
		if !notClosed || v.Interface() == nil {
			break
		}
		valueFunc.Call([]reflect.Value{v})
	}
	return
}

// UnmarshalBytesToCallback parses the CSV from the bytes and send each value to the given func f.
// The func must look like func(Struct).
func UnmarshalBytesToCallback(in []byte, f interface{}) (err error) {
	return UnmarshalToCallback(bytes.NewReader(in), f)
}

// UnmarshalStringToCallback parses the CSV from the string and send each value to the given func f.
// The func must look like func(Struct).
func UnmarshalStringToCallback(in string, c interface{}) (err error) {
	return UnmarshalToCallback(strings.NewReader(in), c)
}

func CSVToMap(in io.Reader) (map[string]string, error) {
	decoder := newDecoder(in)
	header, err := decoder.getCSVRow()
	if err != nil {
		return nil, err
	}
	if len(header) != 2 {
		return nil, fmt.Errorf("maps can only be created for csv of two columns")
	}
	m := make(map[string]string)
	for {
		line, err := decoder.getCSVRow()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		m[line[0]] = line[1]
	}
	return m, nil
}