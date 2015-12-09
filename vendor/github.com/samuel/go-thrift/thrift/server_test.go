package thrift

import (
	"bytes"
	"net/rpc"
	"testing"
)

// Make sure the ServerCodec returns the same method name
// in the response as was in the request.
func TestServerMethodName(t *testing.T) {
	buf := &ClosingBuffer{&bytes.Buffer{}}
	clientCodec := NewClientCodec(NewTransport(buf, BinaryProtocol), false)
	defer clientCodec.Close()
	serverCodec := NewServerCodec(NewTransport(buf, BinaryProtocol))
	defer serverCodec.Close()
	req := &rpc.Request{
		ServiceMethod: "some_method",
		Seq:           3,
	}
	empty := &struct{}{}
	if err := clientCodec.WriteRequest(req, empty); err != nil {
		t.Fatal(err)
	}
	var req2 rpc.Request
	if err := serverCodec.ReadRequestHeader(&req2); err != nil {
		t.Fatal(err)
	}
	if req.Seq != req2.Seq {
		t.Fatalf("Expected seq %d, got %d", req.Seq, req2.Seq)
	}
	t.Logf("Mangled method name: %s", req2.ServiceMethod)
	if err := serverCodec.ReadRequestBody(empty); err != nil {
		t.Fatal(err)
	}
	res := &rpc.Response{
		ServiceMethod: req2.ServiceMethod,
		Seq:           req2.Seq,
	}
	if err := serverCodec.WriteResponse(res, empty); err != nil {
		t.Fatal(err)
	}
	var res2 rpc.Response
	if err := clientCodec.ReadResponseHeader(&res2); err != nil {
		t.Fatal(err)
	}
	if res2.Seq != req.Seq {
		t.Fatalf("Expected seq %d, got %d", req.Seq, res2.Seq)
	}
	if res2.Error != "" {
		t.Fatalf("Expected error of '' instead of '%s'", res2.Error)
	}
	if res2.ServiceMethod != req.ServiceMethod {
		t.Fatalf("Expected ServiceMethod of '%s' instead of '%s'", req.ServiceMethod, res2.ServiceMethod)
	}
}
