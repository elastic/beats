package parser

import (
	"io"
)

// Basic string Input for parsing over a string input.
type StringInput struct {
	state  State
	input  []rune
	offset int
	txn    []int
}

func NewStringInput(s string) *StringInput {
	return &StringInput{
		input: []rune(s),
		txn:   make([]int, 0, 8),
	}
}

func (s *StringInput) Begin() {
	s.txn = append(s.txn, s.offset)
}

func (s *StringInput) End(rollback bool) {
	i := s.txn[len(s.txn)-1]
	s.txn = s.txn[:len(s.txn)-1]
	if rollback {
		s.offset = i
	}
}

func (s *StringInput) Get(i int) (string, error) {
	if len(s.input) < s.offset+i {
		return "", io.EOF
	}

	return string(s.input[s.offset : s.offset+i]), nil
}

func (s *StringInput) Next() (rune, error) {
	if len(s.input) < s.offset+1 {
		return 0, io.EOF
	}

	return s.input[s.offset], nil
}

func (s *StringInput) Pop(i int) {
	s.offset += i
}

func (s *StringInput) Position() Position {
	return Position{Offset: s.offset}
}
