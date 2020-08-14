// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

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
	"os"
	"strings"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow"
)

var (
	outputFile = flag.String("output", "zfields.go", "Output file")
	nameCol    = flag.Int("column-name", 0, "Index of column with field name")
	typeCol    = flag.Int("column-type", 0, "Index of column with field type")
	indent     = flag.Int("indent", 0, "Number of spaces to indent")
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

var indentString string

func makeIndent(n int) (s []byte) {
	if n > 0 {
		s = make([]byte, n)
		for i := 0; i < n; i++ {
			s[i] = ' '
		}
	}
	return s
}

func write(w io.Writer, msg string) {
	for _, line := range strings.Split(msg, "\n") {
		writeLine(w, indentString+line+"\n")
	}
}

func writeLine(w io.Writer, line string) {
	if n, err := w.Write([]byte(line)); err != nil || n != len(line) {
		fmt.Fprintf(os.Stderr, "Failed writing to %s: %v\n", *outputFile, err)
		os.Exit(4)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: fields_gen [-output file.yml] [--column-{name|type}=N]* <input.csv>\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func requireColumn(colFlag *int, argument string) {
	if *colFlag <= 0 {
		fmt.Fprintf(os.Stderr, "Required argument %s not provided\n", argument)
		usage()
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Fprintf(os.Stderr, "No CSV file to parse provided\n")
		usage()
	}
	csvFile := flag.Args()[0]
	if len(csvFile) == 0 {
		fmt.Fprintf(os.Stderr, "Argument -input is required\n")
		os.Exit(2)
	}

	requireColumn(nameCol, "--column-name")
	requireColumn(typeCol, "--column-type")

	indentString = string(makeIndent(*indent))

	fHandle, err := os.Open(csvFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", csvFile, err)
		os.Exit(2)
	}
	defer fHandle.Close()

	outHandle, err := os.Create(*outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create %s: %v\n", *outputFile, err)
		os.Exit(3)
	}
	defer outHandle.Close()

	headerHandle, err := os.Open(*header)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", *header, err)
		os.Exit(2)
	}
	defer headerHandle.Close()

	fileHeader, err := ioutil.ReadAll(headerHandle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", *header, err)
		os.Exit(2)
	}
	write(outHandle, string(fileHeader))

	filtered := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(fHandle)
	for scanner.Scan() {
		if len(scanner.Bytes()) == 0 || scanner.Bytes()[0] != ';' {
			filtered.Write(scanner.Bytes())
			filtered.WriteByte('\n')
		}
	}
	reader := csv.NewReader(filtered)
	for lineNum := 1; ; lineNum++ {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Fprintf(os.Stderr, "read of %s failed: %v\n", csvFile, err)
			os.Exit(5)
		}
		n := len(record)
		vars := make(map[string]string)
		for _, f := range []struct {
			column int
			name   string
		}{
			{*nameCol, "name"},
			{*typeCol, "type"},
		} {
			if f.column > 0 {
				if f.column > n {
					fmt.Fprintf(os.Stderr, "%s column is out of range in line %d\n", f.name, lineNum)
					os.Exit(6)
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
		write(outHandle, fmt.Sprintf(`        - name: %s
          type: %s
`,
			netflow.CamelCaseToSnakeCase(vars["name"]), esType))
	}
}
