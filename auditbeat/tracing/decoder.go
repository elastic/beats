// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package tracing

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

// This sets a limit on struct decoder's greedy fields. 2048 is not a problem
// as the kernel tracing subsystem won't let us dump more than that anyway.
const maxRawCopySize = 2048

// Decoder decodes raw events into an usable type.
type Decoder interface {
	// Decode takes a raw message and its metadata and returns a representation
	// in a decoder-dependent type.
	Decode(raw []byte, meta Metadata) (interface{}, error)
}

type mapDecoder []Field

// NewMapDecoder creates a new decoder that will parse raw tracing events
// into a map[string]interface{}. This decoder will decode all the fields
// described in the format.
// The map keys are the field names as given in the format.
// The map values are fixed-size integers for integer fields:
// uint8, uint16, uint32, uint64, and for signed fields, their signed counterpart.
// For string fields, the value is a string.
// Null string fields will be the null interface.
func NewMapDecoder(format ProbeFormat) Decoder {
	fields := make([]Field, 0, len(format.Fields))
	for _, field := range format.Fields {
		fields = append(fields, field)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Offset < fields[j].Offset
	})
	return mapDecoder(fields)
}

// Decode implements the Decoder interface.
func (f mapDecoder) Decode(raw []byte, meta Metadata) (mapIf interface{}, err error) {
	n := len(raw)
	m := make(map[string]interface{}, len(f)+1)
	m["meta"] = meta
	for _, field := range f {
		if field.Offset+field.Size > n {
			return nil, fmt.Errorf("perf event field %s overflows message of size %d", field.Name, n)
		}
		var value interface{}
		ptr := unsafe.Pointer(&raw[field.Offset])
		switch field.Type {
		case FieldTypeInteger:
			if value, err = readInt(ptr, uint8(field.Size), field.Signed); err != nil {
				return nil, fmt.Errorf("bad size=%d for integer field=%s", field.Size, field.Name)
			}

		case FieldTypeString:
			offset := int(MachineEndian.Uint16(raw[field.Offset:]))
			len := int(MachineEndian.Uint16(raw[field.Offset+2:]))
			if offset+len > n {
				return nil, fmt.Errorf("perf event string data for field %s overflows message of size %d", field.Name, n)
			}
			// (null) strings have data offset equal to string description offset
			if len != 0 || offset != field.Offset {
				if len > 0 && raw[offset+len-1] == 0 {
					len--
				}
				value = string(raw[offset : offset+len])
			}
		}
		m[field.Name] = value
	}
	return m, nil
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
// The special greedy modifier allows a field to capture the named argument
// ("dump" in this case) and all the following unnamed arguments after it.
// This allows to dump chunks of memory by concatenating successive fetchargs
// arguments.
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

	var inFieldsByOffset map[int]Field

	for i := 0; i < tSample.NumField(); i++ {
		outField := tSample.Field(i)
		values, found := outField.Tag.Lookup("kprobe")
		if !found {
			// Untagged field
			continue
		}

		var name string
		var allowUndefined bool
		var greedy bool
		for idx, param := range strings.Split(values, ",") {
			switch param {
			case "allowundefined":
				// it is okay not to find it in the desc.Fields
				allowUndefined = true
			case "greedy":
				greedy = true
			default:
				if idx != 0 {
					return nil, fmt.Errorf("bad parameter '%s' in kprobe tag for field '%s'", param, outField.Name)
				}
				name = param
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
			if allowUndefined {
				continue
			}
			return nil, fmt.Errorf("field '%s' not found in kprobe format description", name)
		}

		if greedy {
			// When greedy is used for the first time, build a map of kprobe's
			// fields by the offset they appear.
			if inFieldsByOffset == nil {
				inFieldsByOffset = make(map[int]Field)
				for _, v := range desc.Fields {
					inFieldsByOffset[v.Offset] = v
				}
			}

			greedySize := uintptr(inField.Size)
			nextOffset := inField.Offset + inField.Size
			nextFieldID := -1
			for {
				nextField, found := inFieldsByOffset[nextOffset]
				if !found {
					break
				}
				if strings.Index(nextField.Name, "arg") != 0 {
					break
				}
				fieldID, err := strconv.Atoi(nextField.Name[3:])
				if err != nil {
					break
				}
				if nextFieldID != -1 && nextFieldID != fieldID {
					break
				}
				greedySize += uintptr(nextField.Size)
				nextOffset += nextField.Size
				nextFieldID = fieldID + 1
			}

			if greedySize > maxRawCopySize {
				return nil, fmt.Errorf("greedy field '%s' exceeds limit of %d bytes", outField.Name, maxRawCopySize)
			}
			if curSize := outField.Type.Size(); curSize != greedySize {
				return nil, fmt.Errorf("greedy field '%s' size is %d but greedy requires %d", outField.Name, curSize, greedySize)
			}

			dec.fields = append(dec.fields, fieldDecoder{
				name: name,
				typ:  FieldTypeRaw,
				src:  uintptr(inField.Offset),
				dst:  outField.Offset,
				len:  greedySize,
			})
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
			err := copyInt(unsafe.Add(destPtr, dec.dst), unsafe.Pointer(&raw[dec.src]), uint8(dec.len))
			if err != nil {
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

type dumpDecoder struct {
	start int
	end   int
}

// NewDumpDecoder returns a new decoder that will dump all the arguments
// as a byte slice. Useful for memory dumps. Arguments must be:
// - unnamed, so they get an automatic argNN name.
// - integer of 64bit (u64 / s64).
// - dump consecutive memory.
func NewDumpDecoder(format ProbeFormat) (Decoder, error) {
	fields := make([]Field, 0, len(format.Fields))

	for name, field := range format.Fields {
		if strings.Index(name, "arg") != 0 {
			continue
		}
		if field.Type != FieldTypeInteger {
			return nil, fmt.Errorf("field '%s' is not an integer", name)
		}
		if field.Size != 8 {
			return nil, fmt.Errorf("field '%s' length is not 8", name)
		}
		fields = append(fields, field)
	}
	if len(fields) == 0 {
		return nil, errors.New("no fields to decode")
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Offset < fields[j].Offset
	})

	base := fields[0].Offset
	end := base
	for _, field := range fields {
		if field.Offset != end {
			return nil, fmt.Errorf("gap before field '%s'", field.Name)
		}
		end += field.Size
	}
	return &dumpDecoder{
		start: base,
		end:   end,
	}, nil
}

// Decode implements the decoder interface.
func (d *dumpDecoder) Decode(raw []byte, _ Metadata) (interface{}, error) {
	if len(raw) < d.end {
		return nil, errors.New("record too short for dump")
	}
	return append([]byte(nil), raw[d.start:d.end]...), nil
}
