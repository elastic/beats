// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow"
)

var (
	outputFile = flag.String("output", "fields.yml", "Output file")
	header     = flag.String("header", "fields.header.yml", "File with header fields to prepend")
)

// Mapping from NetFlow datatypes to Elasticsearch datatypes
// Types not present are ignored
var typesToElasticTypes = map[string]string{
	"octetarray":           "short",
	"unsigned8":            "short",
	"unsigned16":           "integer",
	"unsigned32":           "long",
	"unsigned64":           "long",
	"signed8":              "byte",
	"signed16":             "short",
	"signed32":             "integer",
	"signed64":             "long",
	"float32":              "float",
	"float64":              "double",
	"boolean":              "boolean",
	"macaddress":           "keyword",
	"string":               "keyword",
	"datetimeseconds":      "date",
	"datetimemilliseconds": "date",
	"datetimemicroseconds": "date",
	"datetimenanoseconds":  "date",
	"ipv4address":          "ip",
	"ipv6address":          "ip",
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: fields_gen [-header=file] [-output=file.yml] [input-csv,name-column,type-column,has-header]+\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "No CSV file args to parse provided\n")
		usage()
	}

	if err := generateFieldsYml(flag.Args()); err != nil {
		log.Fatal(err)
	}
}

func generateFieldsYml(args []string) error {
	// Parse the arguments containing file path and parsing parameters.
	var csvFiles []CSVFile
	for _, v := range flag.Args() {
		csvFile, err := NewCSVFileFromArg(v)
		if err != nil {
			return err
		}
		csvFiles = append(csvFiles, *csvFile)
	}

	// Read in all the field data.
	var allFields []map[string]string
	for _, csvFile := range csvFiles {
		fields, err := csvFile.ReadFields()
		if err != nil {
			return err
		}
		allFields = append(allFields, fields)
	}

	// Merge fields and resolve conflicts in the data types.
	fields, err := mergeFields(allFields...)
	if err != nil {
		return err
	}

	// Sort fields alphabetically by name.
	type netflowField struct {
		Name, Type string
	}
	var sortedFields []netflowField
	for k, v := range fields {
		sortedFields = append(sortedFields, netflowField{k, v})
	}
	sort.Slice(sortedFields, func(i, j int) bool {
		return sortedFields[i].Name < sortedFields[j].Name
	})

	headerHandle, err := os.Open(*header)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", *header, err)
	}
	defer headerHandle.Close()

	fileHeader, err := ioutil.ReadAll(headerHandle)
	if err != nil {
		return fmt.Errorf("failed to read header %s: %v", *header, err)
	}

	outHandle, err := os.Create(*outputFile)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", *outputFile, err)
	}
	defer outHandle.Close()

	out := bufio.NewWriter(outHandle)
	defer out.Flush()

	// Write output file.
	writeLine(out, strings.Repeat("#", 40))
	writeLine(out, "# This file is generated. Do not modify.")
	writeLine(out, strings.Repeat("#", 40))
	writeLine(out, string(fileHeader))

	for _, f := range sortedFields {
		writeLine(out, "        - name: "+f.Name)
		writeLine(out, "          type: "+f.Type)
		writeLine(out, "")
	}
	return nil
}

// CSVFile represents a CSV file with containing netflow field information
// (field name, data type).
type CSVFile struct {
	Path       string
	NameColumn int
	TypeColumn int
	Header     bool
}

func NewCSVFileFromArg(arg string) (*CSVFile, error) {
	r := csv.NewReader(strings.NewReader(arg))
	parts, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to parse argument %q: %w", arg, err)
	}
	if len(parts) != 4 {
		return nil, fmt.Errorf("input argument must consist of 4 parts [path,name-column,type-column,header]")
	}

	a := &CSVFile{}
	a.Path = parts[0]
	if a.NameColumn, err = strconv.Atoi(parts[1]); err != nil {
		return nil, fmt.Errorf("failed to parse name column %q: %w", parts[1], err)
	}
	if a.TypeColumn, err = strconv.Atoi(parts[2]); err != nil {
		return nil, fmt.Errorf("failed to parse type column %q: %w", parts[2], err)
	}
	if a.Header, err = strconv.ParseBool(parts[3]); err != nil {
		return nil, fmt.Errorf("failed to parse header column %q: %w", parts[3], err)
	}
	return a, nil
}

// ReadFields reads the fields contained in the CSV file and returns a map
// of names to Elasticsearch data type.
func (a CSVFile) ReadFields() (map[string]string, error) {
	fHandle, err := os.Open(a.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %v: %w", a.Path, err)
	}
	defer fHandle.Close()

	filtered := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(fHandle)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 || scanner.Bytes()[0] != ';' {
			filtered.Write(scanner.Bytes())
			filtered.WriteByte('\n')
		}
	}
	if scanner.Err() != nil {
		return nil, fmt.Errorf("failed reading from %v: %w", a.Path, err)
	}

	fields := map[string]string{}
	reader := csv.NewReader(filtered)
	for lineNum := 1; ; lineNum++ {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("read of %s failed: %v\n", a.Path, err)
		}

		n := len(record)
		vars := make(map[string]string)
		for _, f := range []struct {
			column int
			name   string
		}{
			{a.NameColumn, "name"},
			{a.TypeColumn, "type"},
		} {
			if f.column > 0 {
				if f.column > n {
					return nil, fmt.Errorf("%s column is out of range in line %d\n", f.name, lineNum)
				}
				vars[f.name] = record[f.column-1]
			}
		}
		if len(vars["type"]) == 0 {
			continue
		}

		esType, found := typesToElasticTypes[strings.ToLower(vars["type"])]
		if !found {
			continue
		}

		fields[netflow.CamelCaseToSnakeCase(vars["name"])] = esType
	}

	return fields, nil
}

func mergeFields(allFields ...map[string]string) (map[string]string, error) {
	out := map[string]string{}
	for _, fields := range allFields {
		for name, esType := range fields {
			if existingESType, found := out[name]; found {
				var err error
				esType, err = resolveConflict(existingESType, esType)
				if err != nil {
					return nil, fmt.Errorf("field %v: %w", name, err)
				}
			}
			out[name] = esType
		}
	}
	return out, nil
}

func resolveConflict(a, b string) (string, error) {
	if a == b {
		// No conflict.
		return a, nil
	}
	if a == "keyword" || b == "keyword" {
		// If either is a keyword then use that.
		return "keyword", nil
	}
	return "", fmt.Errorf("cannot resolve type conflict between %v != %v", a, b)
}

func writeLine(w io.StringWriter, line string) {
	if _, err := w.WriteString(line + "\n"); err != nil {
		log.Fatalf("Failed writing line: %v", err)
	}
}
