# BadIO

[![GoDoc](https://godoc.org/github.com/cavaliercoder/badio?status.svg)](https://godoc.org/github.com/cavaliercoder/badio) [![Build Status](https://travis-ci.org/cavaliercoder/badio.svg?branch=master)](https://travis-ci.org/cavaliercoder/badio) [![Go Report Card](https://goreportcard.com/badge/github.com/cavaliercoder/badio)](https://goreportcard.com/report/github.com/cavaliercoder/badio)

Package badio contains extensions to Go's [testing/iotest](https://golang.org/pkg/testing/iotest/)
package and implements Readers and Writers useful mainly for testing.


## Installation

	$ go get github.com/cavaliercoder/badio


## Example

```go
r := badio.NewSequenceReader([]byte("na"))

p := make([]byte, 20)
r.Read(p)

fmt.Printf("ba%s\n", p)

// Prints: banananananananananana

```

## License

Copyright (c) 2015 Ryan Armstrong

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
