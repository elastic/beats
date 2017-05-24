package k8s

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ericchiang/k8s/runtime"
	"github.com/golang/protobuf/proto"
)

type codec struct {
	contentType string
	marshal     func(interface{}) ([]byte, error)
	unmarshal   func([]byte, interface{}) error
}

var (
	// Kubernetes implements its own custom protobuf format to allow clients (and possibly
	// servers) to use either JSON or protocol buffers. The protocol introduces a custom content
	// type and magic bytes to signal the use of protobufs, and wraps each object with API group,
	// version and resource data.
	//
	// The protocol spec which this client implements can be found here:
	//
	//   https://github.com/kubernetes/kubernetes/blob/master/docs/proposals/protobuf.md
	//
	pbCodec = &codec{
		contentType: "application/vnd.kubernetes.protobuf",
		marshal:     marshalPB,
		unmarshal:   unmarshalPB,
	}
	jsonCodec = &codec{
		contentType: "application/json",
		marshal:     json.Marshal,
		unmarshal:   json.Unmarshal,
	}
)

var magicBytes = []byte{0x6b, 0x38, 0x73, 0x00}

func unmarshalPB(b []byte, obj interface{}) error {
	message, ok := obj.(proto.Message)
	if !ok {
		return fmt.Errorf("expected obj of type proto.Message, got %T", obj)
	}
	if len(b) < len(magicBytes) {
		return errors.New("payload is not a kubernetes protobuf object")
	}
	if !bytes.Equal(b[:len(magicBytes)], magicBytes) {
		return errors.New("payload is not a kubernetes protobuf object")
	}

	u := new(runtime.Unknown)
	if err := u.Unmarshal(b[len(magicBytes):]); err != nil {
		return fmt.Errorf("unmarshal unknown: %v", err)
	}
	return proto.Unmarshal(u.Raw, message)
}

func marshalPB(obj interface{}) ([]byte, error) {
	message, ok := obj.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("expected obj of type proto.Message, got %T", obj)
	}
	payload, err := proto.Marshal(message)
	if err != nil {
		return nil, err
	}

	// The URL path informs the API server what the API group, version, and resource
	// of the object. We don't need to specify it here to talk to the API server.
	body, err := (&runtime.Unknown{Raw: payload}).Marshal()
	if err != nil {
		return nil, err
	}

	d := make([]byte, len(magicBytes)+len(body))
	copy(d[:len(magicBytes)], magicBytes)
	copy(d[len(magicBytes):], body)
	return d, nil
}
