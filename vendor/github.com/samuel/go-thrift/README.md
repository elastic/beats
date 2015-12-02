Thrift Package for Go
=====================

[![Build Status](https://travis-ci.org/samuel/go-thrift.png)](https://travis-ci.org/samuel/go-thrift)

API Documentation: <http://godoc.org/github.com/samuel/go-thrift>

License
-------

3-clause BSD. See LICENSE file.

Overview
--------

Thrift is an IDL that can be used to generate RPC client and server
bindings for a variety of languages. This package includes client and server
codecs, serialization, and code generation for Go. It tries to be a more
natural mapping to the language compared to other implementations. For instance,
Go already has the idea of a thrift transport in the ReadWriteCloser interfaces.

Types
-----

Most types map directly to the native Go types, but there are some
quirks and limitations.

* Go supports a more limited set of types for map keys than Thrift
* To use a set define the field as []type and provide a tag of "set":

        StringSet []string `thrift:"1,set"`

* []byte get encoded/decoded as a string because the Thrift binary type
  is the same as string on the wire.

RPC
---

The standard Go net/rpc package is used to provide RPC. Although, one
incompatibility is the net/rpc's use of ServiceName.Method for naming
RPC methods. To get around this the Thrift ServerCodec prefixes method
names with "Thrift".

### Transport

There are no specific transport "classes" as there are in most Thrift
libraries. Instead, the standard `io.ReadWriteCloser` is used as the
interface. If the value also implements the thrift.Flusher interface
then `Flush() error` is called after `protocol.WriteMessageEnd`.

_Framed transport_ is supported by wrapping a value implementing
`io.ReadWriteCloser` with `thrift.NewFramedReadWriteCloser(value)`

### One-way requests

#### Client

One-way request support needs to be enabled on the RPC codec explicitly.
The reason they're not allowed by default is because the Go RPC package
doesn't actually support one-way requests. To get around this requires
a rather janky hack of using channels to track pending requests in the
codec and faking responses.

#### Server

One-way requests aren't yet implemented on the server side.

Parser & Code Generator
-----------------------

The "parser" subdirectory contains a Thrift IDL parser, and "generator"
contains a Go code generator. It could be extended to include other
languages.

How to use the generator:

    $ go install github.com/samuel/go-thrift/generator

    $ generator --help
    Usage of parsimony: [options] inputfile outputpath
      -go.binarystring=false: Always use string for binary instead of []byte
      -go.json.enumnum=false: For JSON marshal enums by number instead of name
      -go.pointers=false: Make all fields pointers

    $ generator cassandra.thrift $GOPATH/src/

TODO
----

* default values
* oneway requests on the server
