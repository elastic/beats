# jsonschema

[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://opensource.org/licenses/BSD-3-Clause)
[![GoDoc](https://godoc.org/github.com/santhosh-tekuri/jsonschema?status.svg)](https://godoc.org/github.com/santhosh-tekuri/jsonschema)
[![Go Report Card](https://goreportcard.com/badge/github.com/santhosh-tekuri/jsonschema)](https://goreportcard.com/report/github.com/santhosh-tekuri/jsonschema)
[![Build Status](https://travis-ci.org/santhosh-tekuri/jsonschema.svg?branch=master)](https://travis-ci.org/santhosh-tekuri/jsonschema)
[![codecov.io](https://codecov.io/github/santhosh-tekuri/jsonschema/coverage.svg?branch=master)](https://codecov.io/github/santhosh-tekuri/jsonschema?branch=master)

Package jsonschema provides json-schema compilation and validation.

This implementation of JSON Schema, supports draft4, draft6 and draft7.

Passes all tests(including optional) in https://github.com/json-schema/JSON-Schema-Test-Suite

An example of using this package:

```go
schema, err := jsonschema.Compile("schemas/purchaseOrder.json")
if err != nil {
    return err
}
f, err := os.Open("purchaseOrder.json")
if err != nil {
    return err
}
defer f.Close()
if err = schema.Validate(f); err != nil {
    return err
}
```

The schema is compiled against the version specified in `$schema` property.
If `$schema` property is missing, it uses latest draft which currently is draft7.
You can force to use draft4 when `$schema` is missing, as follows:

```go
compiler := jsonschema.NewCompiler()
compler.Draft = jsonschema.Draft4
```

you can also validate go value using `schema.ValidateInterface(interface{})` method.  
but the argument should not be user-defined struct.


This package supports loading json-schema from filePath and fileURL.

To load json-schema from HTTPURL, add following import:

```go
import _ "github.com/santhosh-tekuri/jsonschema/httploader"
```

Loading from urls for other schemes (such as ftp), can be plugged in. see package jsonschema/httploader
for an example

To load json-schema from in-memory:

```go
data := `{"type": "string"}`
url := "sch.json"
compiler := jsonschema.NewCompiler()
if err := compiler.AddResource(url, strings.NewReader(data)); err != nil {
    return err
}
schema, err := compiler.Compile(url)
if err != nil {
    return err
}
f, err := os.Open("doc.json")
if err != nil {
    return err
}
defer f.Close()
if err = schema.Validate(f); err != nil {
    return err
}
```

This package supports json string formats: 
- date-time
- date
- time
- hostname
- email
- ip-address
- ipv4
- ipv6
- uri
- uriref/uri-reference
- regex
- format
- json-pointer
- relative-json-pointer
- uri-template (limited validation)

Developers can register their own formats using package "github.com/santhosh-tekuri/jsonschema/formats".

"base64" contentEncoding is supported. Custom decoders can be registered using package "github.com/santhosh-tekuri/jsonschema/decoders".

"application/json" contentMediaType is supported. Custom mediatypes can be registered using package "github.com/santhosh-tekuri/jsonschema/mediatypes".

## ValidationError

The ValidationError returned by Validate method contains detailed context to understand why and where the error is.

schema.json:
```json
{
      "$ref": "t.json#/definitions/employee"
}
```

t.json:
```json
{
    "definitions": {
        "employee": {
            "type": "string"
        }
    }
}
```

doc.json:
```json
1
```

Validating `doc.json` with `schema.json`, gives following ValidationError:
```
I[#] S[#] doesn't validate with "schema.json#"
  I[#] S[#/$ref] doesn't valide with "t.json#/definitions/employee"
    I[#] S[#/definitions/employee/type] expected string, but got number
```

Here `I` stands for instance document and `S` stands for schema document.  
The json-fragments that caused error in instance and schema documents are represented using json-pointer notation.  
Nested causes are printed with indent.

## CLI

```bash
jv <schema-file> [<json-doc>]...
```

if no `<json-doc>` arguments are passed, it simply validates the `<schema-file>`.

exit-code is 1, if there are any validation errors
