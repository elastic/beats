// Copyright 2012 Samuel Stauffer. All rights reserved.
// Use of this source code is governed by a 3-clause BSD
// license that can be found in the LICENSE file.

package thrift

import (
	"errors"
	"io"
	"net"
	"net/rpc"
)

// Implements rpc.ClientCodec
type clientCodec struct {
	conn           Transport
	onewayRequests chan pendingRequest
	twowayRequests chan pendingRequest
	enableOneway   bool
}

type pendingRequest struct {
	method string
	seq    uint64
}

type oneway interface {
	Oneway() bool
}

var (
	// ErrTooManyPendingRequests is the error when there's too many requests that have been
	// sent that have not yet received responses.
	ErrTooManyPendingRequests = errors.New("thrift.client: too many pending requests")
	// ErrOnewayNotEnabled is the error when trying to make a one-way RPC call but the
	// client was not created with one-way support enabled.
	ErrOnewayNotEnabled = errors.New("thrift.client: one way support not enabled on codec")
)

const maxPendingRequests = 1000

// Dial connects to a Thrift RPC server at the specified network address using the specified protocol.
func Dial(network, address string, framed bool, protocol ProtocolBuilder, supportOnewayRequests bool) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	var c io.ReadWriteCloser = conn
	if framed {
		c = NewFramedReadWriteCloser(conn, DefaultMaxFrameSize)
	}
	codec := &clientCodec{
		conn: NewTransport(c, protocol),
	}
	if supportOnewayRequests {
		codec.enableOneway = true
		codec.onewayRequests = make(chan pendingRequest, maxPendingRequests)
		codec.twowayRequests = make(chan pendingRequest, maxPendingRequests)
	}
	return rpc.NewClientWithCodec(codec), nil
}

// NewClient returns a new rpc.Client to handle requests to the set of
// services at the other end of the connection.
func NewClient(conn Transport, supportOnewayRequests bool) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(conn, supportOnewayRequests))
}

// NewClientCodec returns a new rpc.ClientCodec using Thrift RPC on conn using the specified protocol.
func NewClientCodec(conn Transport, supportOnewayRequests bool) rpc.ClientCodec {
	c := &clientCodec{
		conn: conn,
	}
	if supportOnewayRequests {
		c.enableOneway = true
		c.onewayRequests = make(chan pendingRequest, maxPendingRequests)
		c.twowayRequests = make(chan pendingRequest, maxPendingRequests)
	}
	return c
}

func (c *clientCodec) WriteRequest(request *rpc.Request, thriftStruct interface{}) error {
	if err := c.conn.WriteMessageBegin(request.ServiceMethod, MessageTypeCall, int32(request.Seq)); err != nil {
		return err
	}
	if err := EncodeStruct(c.conn, thriftStruct); err != nil {
		return err
	}
	if err := c.conn.WriteMessageEnd(); err != nil {
		return err
	}
	if err := c.conn.Flush(); err != nil {
		return err
	}
	ow := false
	if o, ok := thriftStruct.(oneway); ok {
		ow = o.Oneway()
	}
	if c.enableOneway {
		var err error
		if ow {
			select {
			case c.onewayRequests <- pendingRequest{request.ServiceMethod, request.Seq}:
			default:
				err = ErrTooManyPendingRequests
			}
		} else {
			select {
			case c.twowayRequests <- pendingRequest{request.ServiceMethod, request.Seq}:
			default:
				err = ErrTooManyPendingRequests
			}
		}
		if err != nil {
			return err
		}
	} else if ow {
		return ErrOnewayNotEnabled
	}

	return nil
}

func (c *clientCodec) ReadResponseHeader(response *rpc.Response) error {
	if c.enableOneway {
		select {
		case ow := <-c.onewayRequests:
			response.ServiceMethod = ow.method
			response.Seq = ow.seq
			return nil
		case _ = <-c.twowayRequests:
		}
	}

	name, messageType, seq, err := c.conn.ReadMessageBegin()
	if err != nil {
		return err
	}
	response.ServiceMethod = name
	response.Seq = uint64(seq)
	if messageType == MessageTypeException {
		exception := &ApplicationException{}
		if err := DecodeStruct(c.conn, exception); err != nil {
			return err
		}
		response.Error = exception.String()
		return c.conn.ReadMessageEnd()
	}
	return nil
}

func (c *clientCodec) ReadResponseBody(thriftStruct interface{}) error {
	if thriftStruct == nil {
		// Should only get called if ReadResponseHeader set the Error value in
		// which case we've already read the body (ApplicationException)
		return nil
	}

	if err := DecodeStruct(c.conn, thriftStruct); err != nil {
		return err
	}

	return c.conn.ReadMessageEnd()
}

func (c *clientCodec) Close() error {
	if cl, ok := c.conn.(io.Closer); ok {
		return cl.Close()
	}
	return nil
}
