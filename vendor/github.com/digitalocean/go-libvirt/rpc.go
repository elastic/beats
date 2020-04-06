// Copyright 2016 The go-libvirt Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package libvirt

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync/atomic"

	"github.com/davecgh/go-xdr/xdr2"
	"github.com/digitalocean/go-libvirt/internal/constants"
)

// ErrUnsupported is returned if a procedure is not supported by libvirt
var ErrUnsupported = errors.New("unsupported procedure requested")

// request and response types
const (
	// Call is used when making calls to the remote server.
	Call = iota

	// Reply indicates a server reply.
	Reply

	// Message is an asynchronous notification.
	Message

	// Stream represents a stream data packet.
	Stream

	// CallWithFDs is used by a client to indicate the request has
	// arguments with file descriptors.
	CallWithFDs

	// ReplyWithFDs is used by a server to indicate the request has
	// arguments with file descriptors.
	ReplyWithFDs
)

// request and response statuses
const (
	// StatusOK is always set for method calls or events.
	// For replies it indicates successful completion of the method.
	// For streams it indicates confirmation of the end of file on the stream.
	StatusOK = iota

	// StatusError for replies indicates that the method call failed
	// and error information is being returned. For streams this indicates
	// that not all data was sent and the stream has aborted.
	StatusError

	// StatusContinue is only used for streams.
	// This indicates that further data packets will be following.
	StatusContinue
)

// header is a libvirt rpc packet header
type header struct {
	// Program identifier
	Program uint32

	// Program version
	Version uint32

	// Remote procedure identifier
	Procedure uint32

	// Call type, e.g., Reply
	Type uint32

	// Call serial number
	Serial uint32

	// Request status, e.g., StatusOK
	Status uint32
}

// packet represents a RPC request or response.
type packet struct {
	// Size of packet, in bytes, including length.
	// Len + Header + Payload
	Len    uint32
	Header header
}

// internal rpc response
type response struct {
	Payload []byte
	Status  uint32
}

// libvirt error response
type libvirtError struct {
	Code     uint32
	DomainID uint32
	Padding  uint8
	Message  string
	Level    uint32
}

func (l *Libvirt) connect() error {
	payload := struct {
		Padding [3]byte
		Name    string
		Flags   uint32
	}{
		Padding: [3]byte{0x1, 0x0, 0x0},
		Name:    "qemu:///system",
		Flags:   0,
	}

	buf, err := encode(&payload)
	if err != nil {
		return err
	}

	// libvirt requires that we call auth-list prior to connecting,
	// event when no authentication is used.
	_, err = l.request(constants.ProcAuthList, constants.Program, buf)
	if err != nil {
		return err
	}

	_, err = l.request(constants.ProcConnectOpen, constants.Program, buf)
	if err != nil {
		return err
	}

	return nil
}

func (l *Libvirt) disconnect() error {
	_, err := l.request(constants.ProcConnectClose, constants.Program, nil)
	return err
}

// listen processes incoming data and routes
// responses to their respective callback handler.
func (l *Libvirt) listen() {
	for {
		// response packet length
		length, err := pktlen(l.r)
		if err != nil {
			// When the underlying connection EOFs or is closed, stop
			// this goroutine
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				return
			}

			// invalid packet
			continue
		}

		// response header
		h, err := extractHeader(l.r)
		if err != nil {
			// invalid packet
			continue
		}

		// payload: packet length minus what was previously read
		size := int(length) - (constants.PacketLengthSize + constants.HeaderSize)
		buf := make([]byte, size)
		_, err = io.ReadFull(l.r, buf)
		if err != nil {
			// invalid packet
			continue
		}

		// route response to caller
		l.route(h, buf)
	}
}

// callback sends rpc responses to their respective caller.
func (l *Libvirt) callback(id uint32, res response) {
	l.cm.Lock()
	c, ok := l.callbacks[id]
	l.cm.Unlock()
	if ok {
		c <- res
	}

	l.deregister(id)
}

// route sends incoming packets to their listeners.
func (l *Libvirt) route(h *header, buf []byte) {
	// route events to their respective listener
	if h.Program == constants.ProgramQEMU && h.Procedure == constants.QEMUDomainMonitorEvent {
		l.stream(buf)
		return
	}

	// send responses to caller
	res := response{
		Payload: buf,
		Status:  h.Status,
	}
	l.callback(h.Serial, res)
}

// serial provides atomic access to the next sequential request serial number.
func (l *Libvirt) serial() uint32 {
	return atomic.AddUint32(&l.s, 1)
}

// stream decodes domain events and sends them
// to the respective event listener.
func (l *Libvirt) stream(buf []byte) {
	e, err := decodeEvent(buf)
	if err != nil {
		// event was malformed, drop.
		return
	}

	// send to event listener
	l.em.Lock()
	c, ok := l.events[e.CallbackID]
	l.em.Unlock()
	if ok {
		c <- e
	}
}

// addStream configures the routing for an event stream.
func (l *Libvirt) addStream(id uint32, stream chan *DomainEvent) {
	l.em.Lock()
	l.events[id] = stream
	l.em.Unlock()
}

// removeStream notifies the libvirt server to stop sending events
// for the provided callback id. Upon successful de-registration the
// callback handler is destroyed.
func (l *Libvirt) removeStream(id uint32) error {
	close(l.events[id])

	payload := struct {
		CallbackID uint32
	}{
		CallbackID: id,
	}

	buf, err := encode(&payload)
	if err != nil {
		return err
	}

	_, err = l.request(constants.QEMUConnectDomainMonitorEventDeregister, constants.ProgramQEMU, buf)
	if err != nil {
		return err
	}

	l.em.Lock()
	delete(l.events, id)
	l.em.Unlock()

	return nil
}

// register configures a method response callback
func (l *Libvirt) register(id uint32, c chan response) {
	l.cm.Lock()
	l.callbacks[id] = c
	l.cm.Unlock()
}

// deregister destroys a method response callback
func (l *Libvirt) deregister(id uint32) {
	l.cm.Lock()
	close(l.callbacks[id])
	delete(l.callbacks, id)
	l.cm.Unlock()
}

// request performs a libvirt RPC request.
// returns response returned by server.
// if response is not OK, decodes error from it and returns it.
func (l *Libvirt) request(proc uint32, program uint32, payload []byte) (response, error) {
	serial := l.serial()
	c := make(chan response)

	l.register(serial, c)

	size := constants.PacketLengthSize + constants.HeaderSize
	if payload != nil {
		size += len(payload)
	}

	p := packet{
		Len: uint32(size),
		Header: header{
			Program:   program,
			Version:   constants.ProtocolVersion,
			Procedure: proc,
			Type:      Call,
			Serial:    serial,
			Status:    StatusOK,
		},
	}

	// write header
	l.mu.Lock()
	defer l.mu.Unlock()
	err := binary.Write(l.w, binary.BigEndian, p)
	if err != nil {
		return response{}, err
	}

	// write payload
	if payload != nil {
		err = binary.Write(l.w, binary.BigEndian, payload)
		if err != nil {
			return response{}, err
		}
	}

	if err := l.w.Flush(); err != nil {
		return response{}, err
	}

	resp := <-c
	if resp.Status != StatusOK {
		return resp, decodeError(resp.Payload)
	}

	return resp, nil
}

// encode XDR encodes the provided data.
func encode(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	_, err := xdr.Marshal(&buf, data)

	return buf.Bytes(), err
}

// decodeError extracts an error message from the provider buffer.
func decodeError(buf []byte) error {
	var e libvirtError

	dec := xdr.NewDecoder(bytes.NewReader(buf))
	_, err := dec.Decode(&e)
	if err != nil {
		return err
	}

	if strings.Contains(e.Message, "unknown procedure") {
		return ErrUnsupported
	}

	return errors.New(e.Message)
}

// decodeEvent extracts an event from the given byte slice.
// Errors encountered will be returned along with a nil event.
func decodeEvent(buf []byte) (*DomainEvent, error) {
	var e DomainEvent

	dec := xdr.NewDecoder(bytes.NewReader(buf))
	_, err := dec.Decode(&e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// pktlen determines the length of an incoming rpc response.
// If an error is encountered reading the provided Reader, the
// error is returned and response length will be 0.
func pktlen(r io.Reader) (uint32, error) {
	buf := make([]byte, constants.PacketLengthSize)

	for n := 0; n < cap(buf); {
		nn, err := r.Read(buf)
		if err != nil {
			return 0, err
		}

		n += nn
	}

	return binary.BigEndian.Uint32(buf), nil
}

// extractHeader returns the decoded header from an incoming response.
func extractHeader(r io.Reader) (*header, error) {
	buf := make([]byte, constants.HeaderSize)

	for n := 0; n < cap(buf); {
		nn, err := r.Read(buf)
		if err != nil {
			return nil, err
		}

		n += nn
	}

	h := &header{
		Program:   binary.BigEndian.Uint32(buf[0:4]),
		Version:   binary.BigEndian.Uint32(buf[4:8]),
		Procedure: binary.BigEndian.Uint32(buf[8:12]),
		Type:      binary.BigEndian.Uint32(buf[12:16]),
		Serial:    binary.BigEndian.Uint32(buf[16:20]),
		Status:    binary.BigEndian.Uint32(buf[20:24]),
	}

	return h, nil
}
