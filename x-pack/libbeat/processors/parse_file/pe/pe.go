// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pe

import (
	"debug/pe"
	"os"

	"github.com/elastic/beats/v7/libbeat/common"
	parserCommon "github.com/elastic/beats/v7/x-pack/libbeat/processors/parse_file/common"
)

var ecsResourceMap = map[string]string{
	"CompanyName":      "company",
	"FileDescription":  "description",
	"FileVersion":      "file_version",
	"OriginalFilename": "original_file_name",
	"ProductName":      "product",
}

type parser struct{}

func (p *parser) Identify(header []byte) bool {
	return len(header) > 1 && header[0] == 0x4D && header[1] == 0x5A
}

func (p *parser) Parse(f *os.File) (common.MapStr, error) {
	var emptyMap common.MapStr
	peFile, err := pe.NewFile(f)
	if err != nil {
		return emptyMap, err
	}

	hash, err := imphash(peFile)
	if err != nil {
		return emptyMap, err
	}

	peMap := common.MapStr{
		"imphash": hash,
	}

	for _, section := range peFile.Sections {
		if section.Name == ".rsrc" {
			data, err := section.Data()
			if err != nil {
				return emptyMap, err
			}
			resources, err := parseDirectory(section.VirtualAddress, data)
			if err != nil {
				return emptyMap, err
			}
			versionInfo, err := getVersionInfoForResources(resources)
			if err != nil {
				return emptyMap, err
			}
			updated, err := addVersionInfo(peMap, versionInfo)
			if err != nil {
				return emptyMap, err
			}
			peMap = updated
		}
	}

	return peMap, nil
}

func addVersionInfo(peMap common.MapStr, info []versionInfo) (common.MapStr, error) {
	for _, vInfo := range info {
		if mapped, ok := ecsResourceMap[vInfo.Name]; ok {
			if _, err := peMap.Put(mapped, vInfo.Value); err != nil {
				return peMap, err
			}
		}
	}
	return peMap, nil
}

// NewParser creates a PE parser
func NewParser() parserCommon.Parser {
	return &parser{}
}
