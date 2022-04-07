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

package thrift

import (
	"fmt"

	"github.com/elastic/beats/v8/libbeat/logp"

	"github.com/samuel/go-thrift/parser"
)

type thriftIdlMethod struct {
	service *parser.Service
	method  *parser.Method

	params     []*string
	exceptions []*string
}

type thriftIdl struct {
	methodsByName map[string]*thriftIdlMethod
}

func fieldsToArrayByID(fields []*parser.Field) []*string {
	if len(fields) == 0 {
		return []*string{}
	}

	max := 0
	for _, field := range fields {
		if field.Id > max {
			max = field.Id
		}
	}

	output := make([]*string, max+1)

	for _, field := range fields {
		if len(field.Name) > 0 {
			output[field.Id] = &field.Name
		}
	}

	return output
}

func buildMethodsMap(thriftFiles map[string]parser.Thrift) map[string]*thriftIdlMethod {
	output := make(map[string]*thriftIdlMethod)

	for _, thrift := range thriftFiles {
		for _, service := range thrift.Services {
			for _, method := range service.Methods {
				if _, exists := output[method.Name]; exists {
					logp.Warn("Thrift IDL: Method %s is defined in more services: %s and %s",
						method.Name, output[method.Name].service.Name, service.Name)
				}
				output[method.Name] = &thriftIdlMethod{
					service:    service,
					method:     method,
					params:     fieldsToArrayByID(method.Arguments),
					exceptions: fieldsToArrayByID(method.Exceptions),
				}
			}
		}
	}

	return output
}

func readFiles(files []string) (map[string]parser.Thrift, error) {
	output := make(map[string]parser.Thrift)

	thriftParser := parser.Parser{}

	for _, file := range files {
		filesMap, _, err := thriftParser.ParseFile(file)
		if err != nil {
			return output, fmt.Errorf("Error parsing Thrift IDL file %s: %s", file, err)
		}

		for fname, parsedFile := range filesMap {
			output[fname] = *parsedFile
		}
	}

	return output, nil
}

func (thriftidl *thriftIdl) findMethod(name string) *thriftIdlMethod {
	return thriftidl.methodsByName[name]
}

func newThriftIdl(idlFiles []string) (*thriftIdl, error) {
	if len(idlFiles) == 0 {
		return nil, nil
	}
	thriftFiles, err := readFiles(idlFiles)
	if err != nil {
		return nil, err
	}

	return &thriftIdl{
		methodsByName: buildMethodsMap(thriftFiles),
	}, nil
}
