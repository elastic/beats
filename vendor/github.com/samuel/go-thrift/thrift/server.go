// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"errors"
	"io"
	"net/rpc"
	"strings"
	"sync"
)

type serverCodec struct {
	conn       Transport
	nameCache  map[string]string // incoming name -> registered name
	methodName map[uint64]string // sequence ID -> method name
	mu         sync.Mutex
}

// ServeConn runs the Thrift RPC server on a single connection. ServeConn blocks,
// serving the connection until the client hangs up. The caller typically invokes
// ServeConn in a go statement.
func ServeConn(conn Transport) {
	rpc.ServeCodec(NewServerCodec(conn))
}

// NewServerCodec returns a new rpc.ServerCodec using Thrift RPC on conn using the specified protocol.
func NewServerCodec(conn Transport) rpc.ServerCodec {
	return &serverCodec{
		conn:       conn,
		nameCache:  make(map[string]string, 8),
		methodName: make(map[uint64]string, 8),
	}
}

func (c *serverCodec) ReadRequestHeader(request *rpc.Request) error {
	name, messageType, seq, err := c.conn.ReadMessageBegin()
	if err != nil {
		return err
	}
	if messageType != MessageTypeCall { // Currently don't support one way
		return errors.New("thrift: expected Call message type")
	}

	// TODO: should use a limited size cache for the nameCache to avoid a possible
	//       memory overflow from nefarious or broken clients
	newName := c.nameCache[name]
	if newName == "" {
		newName = CamelCase(name)
		if !strings.ContainsRune(newName, '.') {
			newName = "Thrift." + newName
		}
		c.nameCache[name] = newName
	}

	c.mu.Lock()
	c.methodName[uint64(seq)] = name
	c.mu.Unlock()

	request.ServiceMethod = newName
	request.Seq = uint64(seq)

	return nil
}

func (c *serverCodec) ReadRequestBody(thriftStruct interface{}) error {
	if thriftStruct == nil {
		if err := SkipValue(c.conn, TypeStruct); err != nil {
			return err
		}
	} else {
		if err := DecodeStruct(c.conn, thriftStruct); err != nil {
			return err
		}
	}
	return c.conn.ReadMessageEnd()
}

func (c *serverCodec) WriteResponse(response *rpc.Response, thriftStruct interface{}) error {
	c.mu.Lock()
	methodName := c.methodName[response.Seq]
	delete(c.methodName, response.Seq)
	c.mu.Unlock()
	response.ServiceMethod = methodName

	mtype := byte(MessageTypeReply)
	if response.Error != "" {
		mtype = MessageTypeException
		etype := int32(ExceptionInternalError)
		if strings.HasPrefix(response.Error, "rpc: can't find") {
			etype = ExceptionUnknownMethod
		}
		thriftStruct = &ApplicationException{response.Error, etype}
	}
	if err := c.conn.WriteMessageBegin(response.ServiceMethod, mtype, int32(response.Seq)); err != nil {
		return err
	}
	if err := EncodeStruct(c.conn, thriftStruct); err != nil {
		return err
	}
	if err := c.conn.WriteMessageEnd(); err != nil {
		return err
	}
	return c.conn.Flush()
}

func (c *serverCodec) Close() error {
	if cl, ok := c.conn.(io.Closer); ok {
		return cl.Close()
	}
	return nil
}
