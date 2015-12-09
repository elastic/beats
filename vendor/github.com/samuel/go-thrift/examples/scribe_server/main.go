package main

import (
	"fmt"
	"net"
	"net/rpc"

	"github.com/samuel/go-thrift/examples/scribe"
	"github.com/samuel/go-thrift/thrift"
)

// implementation

type scribeServiceImplementation int

func (s *scribeServiceImplementation) Log(messages []*scribe.LogEntry) (scribe.ResultCode, error) {
	for _, m := range messages {
		fmt.Printf("MSG: %+v\n", m)
	}
	return scribe.ResultCodeOk, nil
}

func main() {
	scribeService := new(scribeServiceImplementation)
	rpc.RegisterName("Thrift", &scribe.ScribeServer{Implementation: scribeService})

	ln, err := net.Listen("tcp", ":1463")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("ERROR: %+v\n", err)
			continue
		}
		fmt.Printf("New connection %+v\n", conn)
		t := thrift.NewTransport(thrift.NewFramedReadWriteCloser(conn, 0), thrift.BinaryProtocol)
		go rpc.ServeCodec(thrift.NewServerCodec(t))
	}
}
