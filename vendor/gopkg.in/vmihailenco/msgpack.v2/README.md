# MessagePack encoding for Golang

[![Build Status](https://travis-ci.org/vmihailenco/msgpack.svg?branch=v2)](https://travis-ci.org/vmihailenco/msgpack)
[![GoDoc](https://godoc.org/github.com/vmihailenco/msgpack?status.svg)](https://godoc.org/github.com/vmihailenco/msgpack)

Supports:
- Primitives, arrays, maps, structs, time.Time and interface{}.
- Appengine *datastore.Key and datastore.Cursor.
- [CustomEncoder](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#example-CustomEncoder)/CustomDecoder interfaces for custom encoding.
- [Extensions](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#example-RegisterExt) to encode type information.
- Renaming fields via `msgpack:"my_field_name"`.
- Inlining struct fields via `msgpack:",inline"`.
- Omitting empty fields via `msgpack:",omitempty"`.
- [Map keys sorting](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#Encoder.SortMapKeys).
- Encoding/decoding all [structs as arrays](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#Encoder.StructAsArray) or [individual structs](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#example-Marshal--AsArray).
- Simple but very fast and efficient [queries](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#example-Decoder-Query).

API docs: https://godoc.org/gopkg.in/vmihailenco/msgpack.v2.
Examples: https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#pkg-examples.

## Installation

Install:

```shell
go get gopkg.in/vmihailenco/msgpack.v2
```

## Quickstart

```go
func ExampleMarshal() {
	type Item struct {
		Foo string
	}

	b, err := msgpack.Marshal(&Item{Foo: "bar"})
	if err != nil {
		panic(err)
	}

	var item Item
	err = msgpack.Unmarshal(b, &item)
	if err != nil {
		panic(err)
	}
	fmt.Println(item.Foo)
	// Output: bar
}
```

## Benchmark

```
BenchmarkStructVmihailencoMsgpack-4   	  200000	     12814 ns/op	    2128 B/op	      26 allocs/op
BenchmarkStructUgorjiGoMsgpack-4      	  100000	     17678 ns/op	    3616 B/op	      70 allocs/op
BenchmarkStructUgorjiGoCodec-4        	  100000	     19053 ns/op	    7346 B/op	      23 allocs/op
BenchmarkStructJSON-4                 	   20000	     69438 ns/op	    7864 B/op	      26 allocs/op
BenchmarkStructGOB-4                  	   10000	    104331 ns/op	   14664 B/op	     278 allocs/op
```

## Howto

Please go through [examples](https://godoc.org/gopkg.in/vmihailenco/msgpack.v2#pkg-examples) to get an idea how to use this package.

## See also

- [Golang PostgreSQL ORM](https://github.com/go-pg/pg)
- [Golang message task queue](https://github.com/go-msgqueue/msgqueue)
