// This file has been generated from 'stacks.yml', do not edit
package gotype

import (
	"reflect"
	"unsafe"

	structform "github.com/elastic/go-structform"
)

type unfolderStack struct {
	current unfolder
	stack   []unfolder
	stack0  [32]unfolder
}

type reflectValueStack struct {
	current reflect.Value
	stack   []reflect.Value
	stack0  [32]reflect.Value
}

type ptrStack struct {
	current unsafe.Pointer
	stack   []unsafe.Pointer
	stack0  [32]unsafe.Pointer
}

type keyStack struct {
	current string
	stack   []string
	stack0  [32]string
}

type idxStack struct {
	current int
	stack   []int
	stack0  [32]int
}

type structformTypeStack struct {
	current structform.BaseType
	stack   []structform.BaseType
	stack0  [32]structform.BaseType
}

func (s *unfolderStack) init(v unfolder) {
	s.current = v
	s.stack = s.stack0[:0]
}

func (s *unfolderStack) push(v unfolder) {
	s.stack = append(s.stack, s.current)
	s.current = v
}

func (s *unfolderStack) pop() unfolder {
	old := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return old
}

func (s *reflectValueStack) init(v reflect.Value) {
	s.current = v
	s.stack = s.stack0[:0]
}

func (s *reflectValueStack) push(v reflect.Value) {
	s.stack = append(s.stack, s.current)
	s.current = v
}

func (s *reflectValueStack) pop() reflect.Value {
	old := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return old
}

func (s *ptrStack) init() {
	s.current = nil
	s.stack = s.stack0[:0]
}

func (s *ptrStack) push(v unsafe.Pointer) {
	s.stack = append(s.stack, s.current)
	s.current = v
}

func (s *ptrStack) pop() unsafe.Pointer {
	old := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return old
}

func (s *keyStack) init() {
	s.current = ""
	s.stack = s.stack0[:0]
}

func (s *keyStack) push(v string) {
	s.stack = append(s.stack, s.current)
	s.current = v
}

func (s *keyStack) pop() string {
	old := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return old
}

func (s *idxStack) init() {
	s.current = -1
	s.stack = s.stack0[:0]
}

func (s *idxStack) push(v int) {
	s.stack = append(s.stack, s.current)
	s.current = v
}

func (s *idxStack) pop() int {
	old := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return old
}

func (s *structformTypeStack) init() {
	s.current = structform.AnyType
	s.stack = s.stack0[:0]
}

func (s *structformTypeStack) push(v structform.BaseType) {
	s.stack = append(s.stack, s.current)
	s.current = v
}

func (s *structformTypeStack) pop() structform.BaseType {
	old := s.current
	last := len(s.stack) - 1
	s.current = s.stack[last]
	s.stack = s.stack[:last]
	return old
}
