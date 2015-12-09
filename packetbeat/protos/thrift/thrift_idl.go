package thrift

import (
	"fmt"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/samuel/go-thrift/parser"
)

type ThriftIdlMethod struct {
	Service *parser.Service
	Method  *parser.Method

	Params     []*string
	Exceptions []*string
}

type ThriftIdl struct {
	MethodsByName map[string]*ThriftIdlMethod
}

func fieldsToArrayById(fields []*parser.Field) []*string {
	if len(fields) == 0 {
		return []*string{}
	}

	max := 0
	for _, field := range fields {
		if field.Id > max {
			max = field.Id
		}
	}

	output := make([]*string, max+1, max+1)

	for _, field := range fields {
		if len(field.Name) > 0 {
			output[field.Id] = &field.Name
		}
	}

	return output
}

func BuildMethodsMap(thrift_files map[string]parser.Thrift) map[string]*ThriftIdlMethod {

	output := make(map[string]*ThriftIdlMethod)

	for _, thrift := range thrift_files {
		for _, service := range thrift.Services {
			for _, method := range service.Methods {
				if _, exists := output[method.Name]; exists {
					logp.Warn("Thrift IDL: Method %s is defined in more services: %s and %s",
						output[method.Name].Service.Name, service.Name)
				}
				output[method.Name] = &ThriftIdlMethod{
					Service:    service,
					Method:     method,
					Params:     fieldsToArrayById(method.Arguments),
					Exceptions: fieldsToArrayById(method.Exceptions),
				}
			}
		}
	}

	return output
}

func ReadFiles(files []string) (map[string]parser.Thrift, error) {
	output := make(map[string]parser.Thrift)

	thriftParser := parser.Parser{}

	for _, file := range files {
		files_map, _, err := thriftParser.ParseFile(file)
		if err != nil {
			return output, fmt.Errorf("Error parsing Thrift IDL file %s: %s", file, err)
		}

		for fname, parsedFile := range files_map {
			output[fname] = *parsedFile
		}
	}

	return output, nil
}

func (thriftidl *ThriftIdl) FindMethod(name string) *ThriftIdlMethod {
	return thriftidl.MethodsByName[name]
}

func NewThriftIdl(idl_files []string) (*ThriftIdl, error) {

	if len(idl_files) == 0 {
		return nil, nil
	}
	thrift_files, err := ReadFiles(idl_files)
	if err != nil {
		return nil, err
	}

	return &ThriftIdl{
		MethodsByName: BuildMethodsMap(thrift_files),
	}, nil
}
