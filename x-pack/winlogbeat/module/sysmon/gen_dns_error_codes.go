// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// This file is used to generate the error code number to symbolic name mapping
// used in the Sysmon DNS event pipeline. It uses the Microsoft Error Lookup
// Tool to export the errors from winerror.h to CSV that extracts all of the
// error names that begin with "DNS_". It dumps the data to stdout and it can
// then be pasted in to the module.
//
// See https://docs.microsoft.com/en-us/windows/win32/debug/system-error-code-lookup-tool
// for details about the Microsoft Error Lookup Tool.

const (
	microsoftErrorToolURL    = "https://download.microsoft.com/download/4/3/2/432140e8-fb6c-4145-8192-25242838c542/Err_6.4.5/Err_6.4.5.exe"
	microsoftErrorToolSha256 = "88739EC82BA16A0B4A3C83C1DD2FCA6336AD8E2A1E5F1238C085B1E86AB8834A"
)

var includeCodes = []uint64{
	5,
	8,
	13,
	14,
	123,
	1214,
	1223,
	1460,
	4312,
	9560,
	10054,
	10055,
	10060,
}

func main() {
	hash, err := downloadErrorLookupTool()
	if err != nil {
		log.Fatal(err)
	}

	if hash != microsoftErrorToolSha256 {
		log.Fatalf("bad sha256 for exe file: expected=%s, got=%s", microsoftErrorToolSha256, hash)
	}

	r, err := exportErrors()
	if err != nil {
		log.Fatal(err)
	}

	if err := parseCSV(r); err != nil {
		log.Fatal(err)
	}
}

// download MS Error Lookup tool.
func downloadErrorLookupTool() (string, error) {
	resp, err := http.Get("https://download.microsoft.com/download/4/3/2/432140e8-fb6c-4145-8192-25242838c542/Err_6.4.5/Err_6.4.5.exe")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	f, err := os.Create("Err.exe")
	if err != nil {
		return "", err
	}
	defer f.Close()

	sha256Hash := sha256.New()
	body := io.TeeReader(resp.Body, sha256Hash)
	_, err = io.Copy(f, body)
	return strings.ToUpper(hex.EncodeToString(sha256Hash.Sum(nil))), err
}

// exportErrors to CSV by executing Err.exe.
func exportErrors() (io.Reader, error) {
	csvOutput := new(bytes.Buffer)

	cmd := exec.Command("Err.exe", "/winerror.h", "/:outputtoCSV")
	cmd.Stdout = csvOutput
	cmd.Stderr = os.Stderr

	return csvOutput, cmd.Run()
}

// parseCSV parses the CSV and outputs the error codes that begin with "DNS_".
func parseCSV(r io.Reader) error {
	codes := map[uint64]struct{}{}
	for _, ec := range includeCodes {
		codes[ec] = struct{}{}
	}

	csvReader := csv.NewReader(r)
	for {
		fields, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(fields) != 4 {
			return fmt.Errorf("parse error")
		}

		symbolicName := fields[1]
		errorNumber, err := strconv.ParseUint(fields[0], 0, 64)
		if err != nil {
			log.Printf("Ignoring line because %v: %v", err, strings.Join(fields, ","))
			continue
		}

		_, isIncludedCode := codes[errorNumber]
		if isIncludedCode || strings.HasPrefix(symbolicName, "DNS_") {
			fmt.Printf(`"%d": "%s",`+"\n", errorNumber, symbolicName)
		}
	}
	return nil
}
