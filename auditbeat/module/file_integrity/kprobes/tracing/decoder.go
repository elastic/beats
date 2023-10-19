// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux && (arm64 || amd64 || amd64p32 || 386)

package tracing

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"
)

// Decoder decodes raw events into an usable type.
type Decoder interface {
	// Decode takes a raw message and its metadata and returns a representation
	// in a decoder-dependent type.
	Decode(raw []byte, meta Metadata) (interface{}, error)
}

// AllocateFn is the type of a function that allocates a custom struct
// to be used with StructDecoder. This function must return a pointer to
// a struct.
type AllocateFn func() interface{}

type fieldDecoder struct {
	typ  FieldType
	src  uintptr
	dst  uintptr
	len  uintptr
	name string
}

type structDecoder struct {
	alloc  AllocateFn
	fields []fieldDecoder
}

var intFields = map[reflect.Kind]struct{}{
	reflect.Int:     {},
	reflect.Int8:    {},
	reflect.Int16:   {},
	reflect.Int32:   {},
	reflect.Int64:   {},
	reflect.Uint:    {},
	reflect.Uint8:   {},
	reflect.Uint16:  {},
	reflect.Uint32:  {},
	reflect.Uint64:  {},
	reflect.Uintptr: {},
}

const maxIntSizeBytes = 8

// NewStructDecoder creates a new decoder that will parse raw tracing events
// into a struct.
//
// This custom struct has to be annotated so that the required KProbeFormat
// fields are stored in the appropriate struct fields, as in:
//
//	type myStruct struct {
//		Meta   tracing.Metadata `kprobe:"metadata"`
//		Type   uint16           `kprobe:"common_type"`
//		Flags  uint8            `kprobe:"common_flags"`
//		PCount uint8            `kprobe:"common_preempt_count"`
//		PID    uint32           `kprobe:"common_pid"`
//		IP     uint64           `kprobe:"__probe_ip"`
//		Exe    string           `kprobe:"exe"`
//		Fd     uint64           `kprobe:"fd"`
//		Arg3   uint64           `kprobe:"arg3"`
//		Arg4   uint32           `kprobe:"arg4"`
//		Arg5   uint16           `kprobe:"arg5"`
//		Arg6   uint8            `kprobe:"arg6"`
//		Dump   [12]byte         `kprobe:"dump,greedy"
//	}
//
// There's no need to map all fields captured by fetchargs.
//
// The special metadata field allows to store the event metadata in the returned
// struct. This is optional. The receiver field has to be of Metadata type.
//
// The custom allocator has to return a pointer to the struct. There's no actual
// need to allocate a new struct each time, as long as the consumer of a perf
// event channel manages the lifetime of the returned structs. This allows for
// pooling of events to reduce allocations.
//
// This decoder is faster than the map decoder and results in fewer allocations:
// Only string fields need to be allocated, plus the allocation by allocFn.
func NewStructDecoder(desc ProbeFormat, allocFn AllocateFn) (Decoder, error) {
	dec := new(structDecoder)
	dec.alloc = allocFn

	// Validate that allocFn() returns pointers to structs.
	sample := allocFn()
	tSample := reflect.TypeOf(sample)
	if tSample.Kind() != reflect.Ptr {
		return nil, errors.New("allocator function doesn't return a pointer")
	}
	tSample = tSample.Elem()
	if tSample.Kind() != reflect.Struct {
		return nil, errors.New("allocator function doesn't return a pointer to a struct")
	}

	for i := 0; i < tSample.NumField(); i++ {
		outField := tSample.Field(i)
		values, found := outField.Tag.Lookup("kprobe")
		if !found {
			// Untagged field
			continue
		}

		var name string
		var allowUndefined bool

		kprobeTagValues := strings.Split(values, ",")
		if len(kprobeTagValues) > 2 {
			return nil, fmt.Errorf("bad kprobe tag for field '%s'", outField.Name)
		}

		for _, param := range kprobeTagValues {
			switch {
			case param == "allowundefined":
				// it is okay not to find it in the desc.Fields
				allowUndefined = true
			case name == "":
				name = param
			default:
				return nil, fmt.Errorf("bad parameter '%s' in kprobe tag for field '%s'", param, outField.Name)
			}
		}

		// Special handling for metadata field
		if name == "metadata" {
			if outField.Type != reflect.TypeOf(Metadata{}) {
				return nil, errors.New("bad type for meta field")
			}
			dec.fields = append(dec.fields, fieldDecoder{
				name: name,
				typ:  FieldTypeMeta,
				dst:  outField.Offset,
				// src&len are unused, this avoids checking len against actual payload
				src: 0,
				len: 0,
			})
			continue
		}

		inField, found := desc.Fields[name]
		if !found {
			if !allowUndefined {
				return nil, fmt.Errorf("field '%s' not found in kprobe format description", name)
			}
			continue
		}

		switch inField.Type {
		case FieldTypeInteger:
			if _, found := intFields[outField.Type.Kind()]; !found {
				return nil, fmt.Errorf("wrong struct field type for field '%s', fixed size integer required", name)
			}
			if outField.Type.Size() != uintptr(inField.Size) {
				return nil, fmt.Errorf("wrong struct field size for field '%s', got=%d required=%d",
					name, outField.Type.Size(), inField.Size)
			}
			// Paranoid
			if inField.Size > maxIntSizeBytes {
				return nil, fmt.Errorf("fix me: unexpected integer of size %d in field `%s`",
					inField.Size, name)
			}

		case FieldTypeString:
			if outField.Type.Kind() != reflect.String {
				return nil, fmt.Errorf("wrong struct field type for field '%s', it should be string", name)
			}

		default:
			// Should not happen
			return nil, fmt.Errorf("unexpected field type for field '%s'", name)
		}
		dec.fields = append(dec.fields, fieldDecoder{
			typ:  inField.Type,
			src:  uintptr(inField.Offset),
			dst:  outField.Offset,
			len:  uintptr(inField.Size),
			name: name,
		})
	}
	sort.Slice(dec.fields, func(i, j int) bool {
		return dec.fields[i].src < dec.fields[j].src
	})
	return dec, nil
}

// Decode implements the decoder interface.
func (d *structDecoder) Decode(raw []byte, meta Metadata) (s interface{}, err error) {
	n := uintptr(len(raw))

	// Allocate a new struct to fill
	s = d.alloc()

	// Get a raw pointer to the struct
	destPtr := unsafe.Pointer(reflect.ValueOf(s).Pointer())
	for _, dec := range d.fields {
		if dec.src+dec.len > n {
			return nil, fmt.Errorf("perf event field %s overflows message of size %d", dec.name, n)
		}
		switch dec.typ {
		case FieldTypeInteger:
			switch uint8(dec.len) {
			case 1, 2, 4, 8:
				copy(unsafe.Slice((*byte)(unsafe.Add(destPtr, dec.dst)), dec.len), unsafe.Slice(&raw[dec.src], uint8(dec.len)))
			default:
				return nil, fmt.Errorf("bad size=%d for integer field=%s", dec.len, dec.name)
			}
		case FieldTypeString:
			offset := uintptr(MachineEndian.Uint16(raw[dec.src:]))
			length := uintptr(MachineEndian.Uint16(raw[dec.src+2:]))
			if offset+length > n {
				return nil, fmt.Errorf("perf event string data for field %s overflows message of size %d", dec.name, n)
			}
			if length > 0 && raw[offset+length-1] == 0 {
				length--
			}
			*(*string)(unsafe.Add(destPtr, dec.dst)) = string(raw[offset : offset+length])

		case FieldTypeMeta:
			*(*Metadata)(unsafe.Add(destPtr, dec.dst)) = meta

		case FieldTypeRaw:
			copy(unsafe.Slice((*byte)(unsafe.Add(destPtr, dec.dst)), dec.len), raw[dec.src:])
		}
	}

	return s, nil
}
