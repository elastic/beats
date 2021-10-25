// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build ignore
// +build ignore

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/pkg/errors"
)

var outputFile = flag.String("output", "ftd-ecs-mappings.asciidoc", "Output file")

var outputTables = []struct {
	Name string
	IDs  []string
}{
	{
		Name: "Intrusion events",
		IDs:  []string{"430001"},
	},
	{
		Name: "Connection and Security Intelligence events",
		IDs:  []string{"430002", "430003"},
	},
	{
		Name: "File and Malware events",
		IDs:  []string{"430004", "430004"},
	},
}

type idMappings map[string]fieldMappings

type fieldMappings map[string]stringSet

func main() {
	if err := generate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [-output file.yml] <input.csv>\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func generate() error {
	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) == 0 || len(flag.Args()[0]) == 0 {
		return errors.New("no csv file provided")
	}
	csvFile := flag.Args()[0]
	fHandle, err := os.Open(csvFile)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", csvFile, err)
	}
	defer fHandle.Close()

	outHandle, err := os.Create(*outputFile)
	if err != nil {
		return fmt.Errorf("failed to create %s: %v", *outputFile, err)
	}
	defer outHandle.Close()

	mappings, err := loadMappings(fHandle)
	if err != nil {
		return fmt.Errorf("failed to load mappings from '%s': %v", csvFile, err)
	}

	for _, table := range outputTables {
		fieldMap := make(fieldMappings)
		for _, id := range table.IDs {
			fieldMap.merge(mappings[id])
		}
		var fields []string
		for k, v := range fieldMap {
			if len(v) > 0 {
				fields = append(fields, k)
			}
		}
		sort.Strings(fields)
		fmt.Fprintf(outHandle, "Mappings for %s fields:\n", table.Name)
		fmt.Fprintln(outHandle, "[options=\"header\"]")
		fmt.Fprintln(outHandle, "|====================================")
		fmt.Fprintln(outHandle, "| FTD Field | Mapped fields")
		for _, field := range fields {
			fmt.Fprintln(outHandle, "|", field, "|", fieldMap[field].String())
		}
		fmt.Fprintln(outHandle, "|====================================")
		fmt.Fprintln(outHandle)
	}

	return nil
}

func loadMappings(reader io.Reader) (m idMappings, err error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	m = make(idMappings)
	for lineNum := 1; ; lineNum++ {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return m, errors.Wrapf(err, "failed reading line %d", lineNum)
		}
		if len(record) < 3 {
			return m, fmt.Errorf("line %d has unexpected number of columns: %d", lineNum, len(record))
		}
		id := record[1]
		ftdField := record[2]
		if _, found := m[id]; !found {
			m[id] = make(fieldMappings)
		}
		if _, found := m[id][ftdField]; !found {
			m[id][ftdField] = newStringSet(nil)
		}
		m[id][ftdField].merge(newStringSet(record[3:]))
	}
	return m, nil
}

func (m fieldMappings) merge(other fieldMappings) {
	for ftdField, newECS := range other {
		if curECS, found := m[ftdField]; found {
			curECS.merge(newECS)
		} else {
			m[ftdField] = newECS
		}
	}
}
