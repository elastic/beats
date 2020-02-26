# hjson-go

[![Build Status](https://img.shields.io/travis/hjson/hjson-go.svg?style=flat-square)](http://travis-ci.org/hjson/hjson-go)
[![Go Pkg](https://img.shields.io/github/release/hjson/hjson-go.svg?style=flat-square&label=go-pkg)](https://github.com/hjson/hjson-go/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/hjson/hjson-go?style=flat-square)](https://goreportcard.com/report/github.com/hjson/hjson-go)
[![coverage](https://img.shields.io/badge/coverage-ok-brightgreen.svg?style=flat-square)](http://gocover.io/github.com/hjson/hjson-go/)
[![godoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/hjson/hjson-go)

![Hjson Intro](http://hjson.org/hjson1.gif)

```
{
  # specify rate in requests/second (because comments are helpful!)
  rate: 1000

  // prefer c-style comments?
  /* feeling old fashioned? */

  # did you notice that rate doesn't need quotes?
  hey: look ma, no quotes for strings either!

  # best of all
  notice: []
  anything: ?

  # yes, commas are optional!
}
```

The Go implementation of Hjson is based on [hjson-js](https://github.com/hjson/hjson-js). For other platforms see [hjson.org](http://hjson.org).

# Install

Make sure you have a working Go environment. See the [install instructions](http://golang.org/doc/install.html).

1. Get the sources
```bash
$ go get -u github.com/hjson/hjson-go
```
2. Build the **hjson-cli** commandline tool (optional)
```bash
$ cd $(go env GOPATH)/src/github.com/hjson/hjson-go/hjson-cli && go install
$ hjson-cli --version
```
# Usage as command line tool
```
usage: hjson-cli [OPTIONS] [INPUT]
hjson can be used to convert JSON from/to Hjson.

hjson will read the given JSON/Hjson input file or read from stdin.

Options:
  -allowMinusZero
      Allow -0.
  -bracesSameLine
      Print braces on the same line.
  -c  Output as JSON.
  -h  Show this screen.
  -indentBy string
      The indent string. (default "  ")
  -j  Output as formatted JSON.
  -omitRootBraces
      Omit braces at the root.
  -quoteAlways
      Always quote string values.
```

Sample:
- run `hjson-cli test.json > test.hjson` to convert to Hjson
- run `hjson-cli -j test.hjson > test.json` to convert to JSON

# Usage as a GO library

```go

package main

import (
  "github.com/hjson/hjson-go"
  "fmt"
)

func main() {

    // Now let's look at decoding Hjson data into Go
    // values.
    sampleText := []byte(`
    {
        # specify rate in requests/second
        rate: 1000
        array:
        [
            foo
            bar
        ]
    }`)

    // We need to provide a variable where Hjson
    // can put the decoded data.
    var dat map[string]interface{}

    // Decode and a check for errors.
    if err := hjson.Unmarshal(sampleText, &dat); err != nil {
        panic(err)
    }
    fmt.Println(dat)

    // In order to use the values in the decoded map,
    // we'll need to cast them to their appropriate type.

    rate := dat["rate"].(float64)
    fmt.Println(rate)

    array := dat["array"].([]interface{})
    str1 := array[0].(string)
    fmt.Println(str1)


    // To encode to Hjson with default options:
    sampleMap := map[string]int{"apple": 5, "lettuce": 7}
    hjson, _ := hjson.Marshal(sampleMap)
    // this is short for:
    // options := hjson.DefaultOptions()
    // hjson, _ := hjson.MarshalWithOptions(sampleMap, options)
    fmt.Println(string(hjson))
}
```

If you prefer, you can also unmarshal to Go objects by converting to JSON:

```go

package main

import (
  "github.com/hjson/hjson-go"
  "encoding/json"
  "fmt"
)

type Sample struct {
    Rate  int
    Array []string
}

func main() {

    sampleText := []byte(`
    {
        # specify rate in requests/second
        rate: 1000
        array:
        [
            foo
            bar
        ]
    }`)

    // read Hjson
    var dat map[string]interface{}
    hjson.Unmarshal(sampleText, &dat)

    // convert to JSON
    b, _ := json.Marshal(dat)

    // unmarshal
    var sample Sample
    json.Unmarshal(b, &sample)

    fmt.Println(sample.Rate)
    fmt.Println(sample.Array)
}
```

# API

[![godoc](https://godoc.org/github.com/hjson/hjson-go?status.svg)](http://godoc.org/github.com/hjson/hjson-go)

# History

[see releases](https://github.com/hjson/hjson-go/releases)
