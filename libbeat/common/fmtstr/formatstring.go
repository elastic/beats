package fmtstr

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// FormatEvaler evaluates some format.
type FormatEvaler interface {
	// Eval will execute the format and writes the results into
	// the provided output buffer. Returns error on failure.
	Eval(ctx interface{}, out *bytes.Buffer) error
}

// StringFormatter interface extends FormatEvaler adding support for querying
// formatter meta data.
type StringFormatter interface {
	FormatEvaler

	// Run execute the formatter returning the generated string.
	Run(ctx interface{}) (string, error)

	// IsConst returns true, if execution of formatter will always return the
	// same constant string.
	IsConst() bool
}

// VariableOp defines one expansion variable, including operator and parameter.
// variable operations are always introduced by a collon ':'.
// For example the format string %{x:p1:?p2} has 2 variable operations
// (":", "p1") and (":?", "p2"). It's up to concrete format string implementation
// to compile and interpret variable ops.
type VariableOp struct {
	op    string
	param string
}

type constStringFormatter struct {
	s string
}

type execStringFormatter struct {
	evalers []FormatEvaler
}

type formatElement interface {
	compile(ctx *compileCtx) (FormatEvaler, error)
}

type compileCtx struct {
	compileVariable VariableCompiler
}

// VariableCompiler is used to compile a variable expansion into
// an FormatEvaler to be used with the format-string.
type VariableCompiler func(string, []VariableOp) (FormatEvaler, error)

// StringElement implements StringFormatter always returning a constant string.
type StringElement struct {
	s string
}

type variableElement struct {
	field string
	ops   []VariableOp
}

type token struct {
	typ tokenType
	val string
}

type tokenType uint16

type lexer chan token

const (
	tokErr tokenType = iota + 1
	tokString
	tokOpen
	tokClose
	tokOperator
)

var (
	openToken  = token{tokOpen, "%{"}
	closeToken = token{tokClose, "}"}
)

var (
	errNestedVar          = errors.New("format string variables can not be nested")
	errUnexpectedOperator = errors.New("unexpected formatter operator")
	errMissingClose       = errors.New("missing closing '}'")
	errEmptyFormat        = errors.New("empty format expansion")
	errParamsOpsMismatch  = errors.New("more parameters then ops parsed")
)

// Compile compiles an input format string into a StringFormatter. The variable
// compiler `vc` is invoked for every variable expansion found in the input format
// string. Returns error on parse failure or if variable compiler fails.
//
// Variable expansion are enclosed in expansion braces `%{<expansion>}`.
// The `<expansion>` can contain additional parameters separated by ops
// introduced by collons ':'. For example the format string `%{value:v1:?v2}`
// will be parsed into variable expansion on `value` with variable ops
// `[(":", "v1"), (":?", "v2")]`. It's up to the variable compiler to interpret
// content and variable ops.
//
// The back-slash character `\` acts as escape character.
func Compile(in string, vc VariableCompiler) (StringFormatter, error) {
	ctx := &compileCtx{vc}
	return compile(ctx, in)
}

func compile(ctx *compileCtx, in string) (StringFormatter, error) {
	lexer := makeLexer(in)
	defer lexer.Finish()

	// parse format string
	elements, err := parse(lexer)
	if err != nil {
		return nil, err
	}

	// compile elements into evaluators
	evalers := make([]FormatEvaler, len(elements))
	for i := range elements {
		evalers[i], err = elements[i].compile(ctx)
		if err != nil {
			return nil, err
		}
	}
	evalers = optimize(evalers)

	// try to create constant formatter for constant string
	if len(evalers) == 1 {
		if se, ok := evalers[0].(StringElement); ok {
			return constStringFormatter{se.s}, nil
		}
	}

	// create executable string formatter
	fmt := execStringFormatter{
		evalers: evalers,
	}
	return fmt, nil
}

// optimize optimizes the sequence of evaluators by combining consecutive
// StringElement instances into one StringElement
func optimize(in []FormatEvaler) []FormatEvaler {
	out := in[:0]

	var active StringElement
	isActive := false

	for _, evaler := range in {
		se, isString := evaler.(StringElement)
		if !isString {
			if isActive {
				out = append(out, active)
				isActive = false
			}
			out = append(out, evaler)
			continue
		}

		if !isActive {
			active = se
			isActive = true
			continue
		}
		active.s += se.s
	}

	if isActive {
		out = append(out, active)
	}

	return out
}

func (f constStringFormatter) Eval(_ interface{}, out *bytes.Buffer) error {
	_, err := out.WriteString(f.s)
	return err
}

func (f constStringFormatter) Run(_ interface{}) (string, error) {
	return f.s, nil
}

func (f constStringFormatter) IsConst() bool {
	return true
}

func (f execStringFormatter) Eval(ctx interface{}, out *bytes.Buffer) error {
	for _, evaler := range f.evalers {
		if err := evaler.Eval(ctx, out); err != nil {
			return err
		}
	}
	return nil
}

func (f execStringFormatter) Run(ctx interface{}) (string, error) {
	buf := bytes.NewBuffer(nil)
	if err := f.Eval(ctx, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (f execStringFormatter) IsConst() bool {
	return false
}

func (e StringElement) compile(ctx *compileCtx) (FormatEvaler, error) {
	return e, nil
}

// Eval write the string elements constant string value into
// output buffer.
func (e StringElement) Eval(_ interface{}, out *bytes.Buffer) error {
	_, err := out.WriteString(e.s)
	return err
}

func makeVariableElement(f string, ops, params []string) (variableElement, error) {
	if len(params) > len(ops) {
		return variableElement{}, errParamsOpsMismatch
	}

	out := make([]VariableOp, len(ops))
	for i := range params {
		out[i] = VariableOp{op: ops[i], param: params[i]}
	}
	if len(ops) > len(params) {
		i := len(ops) - 1
		out[i] = VariableOp{op: ops[i]}
	}

	return variableElement{field: f, ops: out}, nil
}

func (e variableElement) compile(ctx *compileCtx) (FormatEvaler, error) {
	return ctx.compileVariable(e.field, e.ops)
}

func parse(lex lexer) ([]formatElement, error) {
	var elems []formatElement

	for token := range lex.Tokens() {
		switch token.typ {
		case tokErr:
			return nil, errors.New(token.val)

		case tokString:
			elems = append(elems, StringElement{token.val})

		case tokOpen:
			elem, err := parseVariable(lex)
			if err != nil {
				return nil, err
			}
			elems = append(elems, elem)

		case tokClose, tokOperator:
			// should not happen, but let's return error just in case
			return nil, fmt.Errorf("Token '%v'(%v) not allowed", token.val, token.typ)
		}
	}

	return elems, nil
}

func parseVariable(lex lexer) (formatElement, error) {
	var strings []string
	var ops []string

	for token := range lex.Tokens() {
		switch token.typ {
		case tokErr:
			return nil, errors.New(token.val)

		case tokOpen:
			return nil, errNestedVar

		case tokClose:
			if len(strings) == 0 {
				return nil, errEmptyFormat
			}
			return makeVariableElement(strings[0], ops, strings[1:])

		case tokString:
			if len(strings) != len(ops) {
				return nil, fmt.Errorf("Unexpected string token %v, expected operator", token.val)
			}
			strings = append(strings, token.val)

		case tokOperator:
			if len(strings) == 0 {
				return nil, errUnexpectedOperator
			}
			ops = append(ops, token.val)
			if len(ops) > len(strings) {
				return nil, fmt.Errorf("Consecutive operator tokens '%v'", token.val)
			}

		default:
			return nil, fmt.Errorf("Unexpected token '%v' (%v)", token.val, token.typ)
		}
	}

	return nil, errMissingClose
}

func makeLexer(in string) lexer {
	lex := make(chan token, 1)

	go func() {
		off := 0
		content := in

		defer func() {
			if len(content) > 0 {
				lex <- token{tokString, content}
			}
			close(lex)
		}()

		strToken := func(s string) {
			if s != "" {
				lex <- token{tokString, s}
			}
		}

		opToken := func(op string) token {
			return token{tokOperator, op}
		}

		varcount := 0
		for len(content) > 0 {
			idx := -1
			if varcount == 0 {
				idx = strings.IndexAny(content[off:], `%\`)
			} else {
				idx = strings.IndexAny(content[off:], `%:}\`)
			}

			if idx == -1 {
				return
			}

			idx += off
			off = idx + 1

			switch content[idx] {
			case '\\': // escape next character
				content = content[:idx] + content[off:]
				continue

			case ':':
				if len(content) <= off { // found ':' at end of string
					return
				}

				strToken(content[:idx])
				op := ":"
				if strings.ContainsRune("!@#&*=+<>?", rune(content[off])) {
					off++
					op = content[idx : off+1]
				}
				lex <- opToken(op)

			case '}':
				strToken(content[:idx])
				lex <- closeToken
				varcount--

			case '%':
				if len(content) <= off { // found '%' at end of string
					return
				}

				if content[off] != '{' {
					continue // no variable expression
				}

				strToken(content[:idx])
				lex <- openToken
				off++
				varcount++
			}

			content = content[off:]
			off = 0
		}

	}()

	return lex
}

func (l lexer) Tokens() <-chan token {
	return (chan token)(l)
}

func (l lexer) Finish() {
	for range l.Tokens() {
	}
}
