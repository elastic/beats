package cfgfile

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Inspired by: https://cuddle.googlecode.com/hg/talk/lex.html

const (
	errUnterminatedBrace = "unterminated brace"
)

// item represents a token returned from the scanner.
type item struct {
	typ itemType // Token type, such as itemVariable.
	pos int      // The starting position, in bytes, of this item in the input string.
	val string   // Value, such as "${".
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	default:
		return i.val
	}
}

// itemType identifies the type of lex items.
type itemType int

// lex tokens.
const (
	itemError itemType = iota + 1
	itemEscapedLeftDelim
	itemLeftDelim
	itemVariable
	itemDefaultValue
	itemRightDelim
	itemText
	itemEOF
)

const eof = -1

// stateFn represents the state of the scanner as a function that returns the
// next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	name    string    // used only for error reports.
	input   string    // the string being scanned.
	start   int       // start position of this item.
	pos     int       // current position in the input.
	width   int       // width of last rune read from input.
	lastPos int       // position of most recent item returned by nextItem
	items   chan item // channel of scanned items.
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.start, l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{itemError, l.start, fmt.Sprintf(format, args...)}
	return nil
}

// nextItem returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

// run lexes the input by executing state functions until the state is nil.
func (l *lexer) run() {
	for state := lexText; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered.
}

// state functions

// token values.
const (
	leftDelim             = "${"
	rightDelim            = '}'
	defaultValueSeperator = ':'
	escapedLeftDelim      = "$${"
)

// lexText scans until an opening action delimiter, "${".
func lexText(l *lexer) stateFn {
	for {
		switch {
		case strings.HasPrefix(l.input[l.pos:], escapedLeftDelim):
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexEscapedLeftDelim
		case strings.HasPrefix(l.input[l.pos:], leftDelim):
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLeftDelim
		}

		if l.next() == eof {
			break
		}
	}
	// Correctly reached EOF.
	if l.pos > l.start {
		l.emit(itemText)
	}
	l.emit(itemEOF)
	return nil
}

// lexEscapedLeftDelim scans the escaped left delimiter, which is known to be
// present.
func lexEscapedLeftDelim(l *lexer) stateFn {
	l.pos += len(escapedLeftDelim)
	l.emit(itemEscapedLeftDelim)
	return lexText
}

// lexLeftDelim scans the left delimiter, which is known to be present.
func lexLeftDelim(l *lexer) stateFn {
	l.pos += len(leftDelim)
	l.emit(itemLeftDelim)
	return lexVariable
}

// lexVariable scans a shell variable name which is alphanumeric and does not
// start with a number or other special shell variable character.
// The ${ has already been scanned.
func lexVariable(l *lexer) stateFn {
	var r rune = l.peek()
	if isShellSpecialVar(r) {
		return l.errorf("shell variable cannot start with %#U", r)
	}
	for {
		r = l.next()
		if !isAlphaNumeric(r) {
			l.backup()
			break
		}
	}
	l.emit(itemVariable)
	return lexDefaultValueOrRightDelim
}

// lexDefaultValueOrRightDelim scans for a default value for the variable
// expansion or for the '}' to close the variable definition.
func lexDefaultValueOrRightDelim(l *lexer) stateFn {
	switch r := l.next(); {
	case r == eof || isEndOfLine(r):
		return l.errorf(errUnterminatedBrace)
	case r == ':':
		l.ignore()
		return lexDefaultValue
	case r == '}':
		l.backup()
		return lexRightDelim
	default:
		return l.errorf("unexpected character in variable expression: %#U, "+
			"expected a default value or closing brace", r)
	}
}

// lexRightDelim scans the right delimiter, which is known to be present.
func lexRightDelim(l *lexer) stateFn {
	l.pos += 1
	l.emit(itemRightDelim)
	return lexText
}

// lexDefaultValue scans the default value for a variable expansion. It scans
// until a '}' is encountered. If EOF or EOL occur before the '}' then this
// is an error.
func lexDefaultValue(l *lexer) stateFn {
loop:
	for {
		r := l.next()
		switch {
		case r == eof || isEndOfLine(r):
			return l.errorf(errUnterminatedBrace)
		case r == rightDelim:
			l.backup()
			break loop
		}
	}
	l.emit(itemDefaultValue)
	return lexRightDelim
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// isShellSpecialVar reports whether r identifies a special shell variable
// such as $*.
func isShellSpecialVar(r rune) bool {
	switch r {
	case '*', '#', '$', '@', '!', '?', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// Functions for using the lexer to parse.

// lex creates a new scanner for the input string.
func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan item),
	}
	go l.run() // Concurrently run state machine.
	return l
}

// parseLexer parses the tokens from the lexer. It expands the environment
// variables that it encounters.
func parseLexer(l *lexer) ([]byte, error) {
	var peekItem *item
	next := func() item {
		if peekItem != nil {
			rtn := *peekItem
			peekItem = nil
			return rtn
		}
		return l.nextItem()
	}
	peek := func() item {
		if peekItem != nil {
			return *peekItem
		}
		rtn := l.nextItem()
		peekItem = &rtn
		return rtn
	}

	var buf bytes.Buffer
loop:
	for {
		item := next()

		switch item.typ {
		case itemText:
			buf.WriteString(item.val)
		case itemVariable:
			variable := item.val
			value := os.Getenv(variable)
			if peek().typ == itemDefaultValue {
				item = next()
				if value == "" {
					value = item.val
				}
			}
			buf.WriteString(value)
		case itemEscapedLeftDelim:
			buf.WriteString(leftDelim)
		case itemLeftDelim, itemRightDelim:
		case itemError:
			return nil, fmt.Errorf("failure while expanding environment "+
				"variables in %s at line=%d, %v", l.name, l.lineNumber(),
				item.val)
		case itemEOF:
			break loop
		default:
			return nil, fmt.Errorf("unexpected token type %d", item.typ)
		}
	}
	return buf.Bytes(), nil
}

// expandEnv replaces ${var} in config according to the values of the current
// environment variables. The replacement is case-sensitive. References to
// undefined variables are replaced by the empty string. A default value can be
// given by using the form ${var:default value}.
//
// Valid variable names consist of letters, numbers, and underscores and do not
// begin with numbers. Variable blocks cannot be split across lines. Unmatched
// braces will causes a parse error. To use a literal '${' in config write
// '$${'.
func expandEnv(filename string, contents []byte) ([]byte, error) {
	l := lex(filename, string(contents))
	return parseLexer(l)
}
