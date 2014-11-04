package main

import (
	"fmt"

	"github.com/samuel/go-thrift/parser"
)

type ThriftIdlMethod struct {
	Service *parser.Service
	Method  *parser.Method
}

func BuildMethodsMap(thrift_files map[string]*parser.Thrift) map[string]ThriftIdlMethod {

	output := make(map[string]ThriftIdlMethod)

	for _, thrift := range thrift_files {
		for _, service := range thrift.Services {
			for _, method := range service.Methods {
				if _, exists := output[method.Name]; exists {
					WARN("Thrift IDL: Method %s is defined in more services: %s and %s",
						output[method.Name].Service.Name, service.Name)
				}
				output[method.Name] = ThriftIdlMethod{
					Service: service,
					Method:  method,
				}
			}
		}
	}

	return output
}

func ReadFiles(files []string) (map[string]*parser.Thrift, error) {
	output := make(map[string]*parser.Thrift)

	thriftParser := parser.Parser{}

	for _, file := range files {
		files_map, _, err := thriftParser.ParseFile(file)
		if err != nil {
			return output, fmt.Errorf("Error parsing Thrift IDL file %s: %s", file, err)
		}

		for fname, parsedFile := range files_map {
			output[fname] = parsedFile
		}
	}

	return output, nil
}
