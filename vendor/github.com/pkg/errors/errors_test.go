package errors

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		err  string
		want error
	}{
		{"", fmt.Errorf("")},
		{"foo", fmt.Errorf("foo")},
		{"foo", New("foo")},
		{"string with format specifiers: %v", errors.New("string with format specifiers: %v")},
	}

	for _, tt := range tests {
		got := New(tt.err)
		if got.Error() != tt.want.Error() {
			t.Errorf("New.Error(): got: %q, want %q", got, tt.want)
		}
	}
}

func TestWrapNil(t *testing.T) {
	got := Wrap(nil, "no error")
	if got != nil {
		t.Errorf("Wrap(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWrap(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error: EOF"},
		{Wrap(io.EOF, "read error"), "client error", "client error: read error: EOF"},
	}

	for _, tt := range tests {
		got := Wrap(tt.err, tt.message).Error()
		if got != tt.want {
			t.Errorf("Wrap(%v, %q): got: %v, want %v", tt.err, tt.message, got, tt.want)
		}
	}
}

type nilError struct{}

func (nilError) Error() string { return "nil error" }

type causeError struct {
	cause error
}

func (e *causeError) Error() string { return "cause error" }
func (e *causeError) Cause() error  { return e.cause }

func TestCause(t *testing.T) {
	x := New("error")
	tests := []struct {
		err  error
		want error
	}{{
		// nil error is nil
		err:  nil,
		want: nil,
	}, {
		// explicit nil error is nil
		err:  (error)(nil),
		want: nil,
	}, {
		// typed nil is nil
		err:  (*nilError)(nil),
		want: (*nilError)(nil),
	}, {
		// uncaused error is unaffected
		err:  io.EOF,
		want: io.EOF,
	}, {
		// caused error returns cause
		err:  &causeError{cause: io.EOF},
		want: io.EOF,
	}, {
		err:  x, // return from errors.New
		want: x,
	}}

	for i, tt := range tests {
		got := Cause(tt.err)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("test %d: got %#v, want %#v", i+1, got, tt.want)
		}
	}
}

func TestFprint(t *testing.T) {
	x := New("error")
	tests := []struct {
		err  error
		want string
	}{{
		// nil error is nil
		err: nil,
	}, {
		// explicit nil error is nil
		err: (error)(nil),
	}, {
		// uncaused error is unaffected
		err:  io.EOF,
		want: "EOF\n",
	}, {
		// caused error returns cause
		err:  &causeError{cause: io.EOF},
		want: "cause error\nEOF\n",
	}, {
		err:  x, // return from errors.New
		want: "github.com/pkg/errors/errors_test.go:106: error\n",
	}, {
		err:  Wrap(x, "message"),
		want: "github.com/pkg/errors/errors_test.go:128: message\ngithub.com/pkg/errors/errors_test.go:106: error\n",
	}, {
		err:  Wrap(Wrap(x, "message"), "another message"),
		want: "github.com/pkg/errors/errors_test.go:131: another message\ngithub.com/pkg/errors/errors_test.go:131: message\ngithub.com/pkg/errors/errors_test.go:106: error\n",
	}, {
		err:  Wrapf(x, "message"),
		want: "github.com/pkg/errors/errors_test.go:134: message\ngithub.com/pkg/errors/errors_test.go:106: error\n",
	}}

	for i, tt := range tests {
		var w bytes.Buffer
		Fprint(&w, tt.err)
		got := w.String()
		if got != tt.want {
			t.Errorf("test %d: Fprint(w, %q): got %q, want %q", i+1, tt.err, got, tt.want)
		}
	}
}

func TestWrapfNil(t *testing.T) {
	got := Wrapf(nil, "no error")
	if got != nil {
		t.Errorf("Wrapf(nil, \"no error\"): got %#v, expected nil", got)
	}
}

func TestWrapf(t *testing.T) {
	tests := []struct {
		err     error
		message string
		want    string
	}{
		{io.EOF, "read error", "read error: EOF"},
		{Wrapf(io.EOF, "read error without format specifiers"), "client error", "client error: read error without format specifiers: EOF"},
		{Wrapf(io.EOF, "read error with %d format specifier", 1), "client error", "client error: read error with 1 format specifier: EOF"},
	}

	for _, tt := range tests {
		got := Wrapf(tt.err, tt.message).Error()
		if got != tt.want {
			t.Errorf("Wrapf(%v, %q): got: %v, want %v", tt.err, tt.message, got, tt.want)
		}
	}
}
