package ucfg

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type reference struct {
	Path cfgPath
}

type expansion struct {
	left, right varEvaler
	pathSep     string
}

type expansionSingle struct {
	evaler  varEvaler
	pathSep string
}

type expansionDefault struct{ expansion }
type expansionAlt struct{ expansion }
type expansionErr struct{ expansion }

type splice struct {
	pieces []varEvaler
}

type varEvaler interface {
	eval(cfg *Config, opts *options) (string, error)
}

type constExp string

type token struct {
	typ tokenType
	val string
}

type parseState struct {
	st     int
	isvar  bool
	op     string
	pieces [2][]varEvaler
}

var (
	errUnterminatedBrace = errors.New("unterminated brace")
	errInvalidType       = errors.New("invalid type")
	errEmptyPath         = errors.New("empty path after expansion")
)

type tokenType uint16

const (
	tokOpen tokenType = iota
	tokClose
	tokSep
	tokString

	// parser state
	stLeft  = 0
	stRight = 1

	opDefault     = ":"
	opAlternative = ":+"
	opError       = ":?"
)

var (
	openToken  = token{tokOpen, "${"}
	closeToken = token{tokClose, "}"}

	sepDefToken = token{tokSep, opDefault}
	sepAltToken = token{tokSep, opAlternative}
	sepErrToken = token{tokSep, opError}
)

func newReference(p cfgPath) *reference {
	return &reference{p}
}

func (r *reference) String() string {
	return fmt.Sprintf("${%v}", r.Path)
}

func (r *reference) resolveRef(cfg *Config, opts *options) (value, error) {
	env := opts.env

	if ok := opts.activeFields.AddNew(r.Path.String()); !ok {
		return nil, raiseCyclicErr(r.Path.String())
	}

	var err Error

	for {
		var v value
		cfg = cfgRoot(cfg)
		if cfg == nil {
			return nil, ErrMissing
		}

		v, err = r.Path.GetValue(cfg, opts)
		if err == nil {
			if v == nil {
				break
			}

			return v, nil
		}

		if len(env) == 0 {
			break
		}

		cfg = env[len(env)-1]
		env = env[:len(env)-1]
	}

	return nil, err
}

func (r *reference) resolveEnv(cfg *Config, opts *options) (string, error) {
	var err error

	if len(opts.resolvers) > 0 {
		key := r.Path.String()
		for i := len(opts.resolvers) - 1; i >= 0; i-- {
			var v string
			resolver := opts.resolvers[i]
			v, err = resolver(key)
			if err == nil {
				return v, nil
			}
		}
	}

	return "", err
}

func (r *reference) resolve(cfg *Config, opts *options) (value, error) {
	v, err := r.resolveRef(cfg, opts)
	if v != nil || criticalResolveError(err) {
		return v, err
	}

	previousErr := err

	s, err := r.resolveEnv(cfg, opts)
	if err != nil {
		// TODO(ph): Not everything is an Error, will do some cleanup in another PR.
		if v, ok := previousErr.(Error); ok {
			if v.Reason() == ErrCyclicReference {
				return nil, previousErr
			}
		}
		return nil, err
	}

	if s == "" {
		return nil, nil
	}

	return newString(context{field: r.Path.String()}, nil, s), nil
}

func (r *reference) eval(cfg *Config, opts *options) (string, error) {
	v, err := r.resolve(cfg, opts)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", fmt.Errorf("can not resolve reference: %v", r.Path)
	}
	return v.toString(opts)
}

func (s constExp) eval(*Config, *options) (string, error) {
	return string(s), nil
}

func (s *splice) String() string {
	return fmt.Sprintf("%v", s.pieces)
}

func (s *splice) eval(cfg *Config, opts *options) (string, error) {
	buf := bytes.NewBuffer(nil)
	for _, p := range s.pieces {
		s, err := p.eval(cfg, opts)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}
	return buf.String(), nil
}

func (e *expansion) String() string {
	return fmt.Sprintf("${%v:%v}", e.left, e.right)
}

func (e *expansionSingle) String() string {
	return fmt.Sprintf("${%v}", e.evaler)
}

func (e *expansionSingle) eval(cfg *Config, opts *options) (string, error) {
	path, err := e.evaler.eval(cfg, opts)
	if err != nil {
		return "", err
	}

	ref := newReference(parsePath(path, e.pathSep))
	return ref.eval(cfg, opts)
}

func (e *expansionDefault) eval(cfg *Config, opts *options) (string, error) {
	path, err := e.left.eval(cfg, opts)
	if err != nil || path == "" {
		return e.right.eval(cfg, opts)
	}
	ref := newReference(parsePath(path, e.pathSep))
	v, err := ref.eval(cfg, opts)
	if err != nil || v == "" {
		return e.right.eval(cfg, opts)
	}
	return v, err
}

func (e *expansionAlt) eval(cfg *Config, opts *options) (string, error) {
	path, err := e.left.eval(cfg, opts)
	if err != nil || path == "" {
		return "", nil
	}

	ref := newReference(parsePath(path, e.pathSep))
	tmp, err := ref.resolve(cfg, opts)
	if err != nil || tmp == nil {
		return "", nil
	}

	return e.right.eval(cfg, opts)
}

func (e *expansionErr) eval(cfg *Config, opts *options) (string, error) {
	path, err := e.left.eval(cfg, opts)
	if err == nil && path != "" {
		ref := newReference(parsePath(path, e.pathSep))
		str, err := ref.eval(cfg, opts)
		if err == nil && str != "" {
			return str, nil
		}
	}

	errStr, err := e.right.eval(cfg, opts)
	if err != nil {
		return "", err
	}
	return "", errors.New(errStr)
}

func (st parseState) finalize(pathSep string) (varEvaler, error) {
	if !st.isvar {
		return nil, errors.New("fatal: processing non-variable state")
	}
	if len(st.pieces[stLeft]) == 0 {
		return nil, errors.New("empty expansion")
	}

	if st.st == stLeft {
		pieces := st.pieces[stLeft]

		if len(pieces) == 0 {
			return constExp(""), nil
		}

		if len(pieces) == 1 {
			if str, ok := pieces[0].(constExp); ok {
				return newReference(parsePath(string(str), pathSep)), nil
			}
		}

		return &expansionSingle{&splice{pieces}, pathSep}, nil
	}

	extract := func(pieces []varEvaler) varEvaler {
		switch len(pieces) {
		case 0:
			return constExp("")
		case 1:
			return pieces[0]
		default:
			return &splice{pieces}
		}
	}

	left := extract(st.pieces[stLeft])
	right := extract(st.pieces[stRight])
	return makeOpExpansion(left, right, st.op, pathSep), nil
}

func makeOpExpansion(l, r varEvaler, op, pathSep string) varEvaler {
	exp := expansion{l, r, pathSep}
	switch op {
	case opDefault:
		return &expansionDefault{exp}
	case opAlternative:
		return &expansionAlt{exp}
	case opError:
		return &expansionErr{exp}
	}
	panic(fmt.Sprintf("Unknown operator: %v", op))
}

func parseSplice(in, pathSep string) (varEvaler, error) {
	lex, errs := lexer(in)
	drainLex := func() {
		for range lex {
		}
	}

	// drain lexer on return so go-routine won't leak
	defer drainLex()

	pieces, perr := parseVarExp(lex, pathSep)
	if perr != nil {
		return nil, perr
	}

	// check for lexer errors
	select {
	case err := <-errs:
		if err != nil {
			return nil, err
		}
	default:
	}

	// return parser result
	return pieces, perr
}

func lexer(in string) (<-chan token, <-chan error) {
	lex := make(chan token, 1)
	errors := make(chan error, 1)

	go func() {
		off := 0
		content := in

		defer func() {
			if len(content) > 0 {
				lex <- token{tokString, content}
			}
			close(lex)
			close(errors)
		}()

		strToken := func(s string) {
			if s != "" {
				lex <- token{tokString, s}
			}
		}

		varcount := 0
		for len(content) > 0 {
			idx := -1
			if varcount == 0 {
				idx = strings.IndexAny(content[off:], "$")
			} else {
				idx = strings.IndexAny(content[off:], "$:}")
			}
			if idx < 0 {
				return
			}

			idx += off
			off = idx + 1
			switch content[idx] {
			case ':':
				if len(content) <= off { // found ':' at end of string
					return
				}

				strToken(content[:idx])
				switch content[off] {
				case '+':
					off++
					lex <- sepAltToken
				case '?':
					off++
					lex <- sepErrToken
				default:
					lex <- sepDefToken
				}

			case '}':
				strToken(content[:idx])
				lex <- closeToken
				varcount--

			case '$':
				if len(content) <= off { // found '$' at end of string
					return
				}

				switch content[off] {
				case '{': // start variable
					strToken(content[:idx])
					lex <- openToken
					off++
					varcount++
				default: // escape any symbol
					content = content[:idx] + content[off:]
					continue
				}
			}

			content = content[off:]
			off = 0
		}
	}()

	return lex, errors
}

func parseVarExp(lex <-chan token, pathSep string) (varEvaler, error) {
	stack := []parseState{{st: stLeft}}

	// parser loop
	for tok := range lex {
		switch tok.typ {
		case tokOpen:
			stack = append(stack, parseState{st: stLeft, isvar: true})
		case tokClose:
			// finalize and pop state
			piece, err := stack[len(stack)-1].finalize(pathSep)
			stack = stack[:len(stack)-1]
			if err != nil {
				return nil, err
			}

			// append result top stacked state
			st := &stack[len(stack)-1]
			st.pieces[st.st] = append(st.pieces[st.st], piece)

		case tokSep: // switch from left to right
			st := &stack[len(stack)-1]
			if !st.isvar {
				return nil, errors.New("default separator not within expansion")
			}
			if st.st == stRight {
				st.pieces[st.st] = addString(st.pieces[st.st], tok.val)
			} else {
				// switch to 'right'
				st.st = stRight
				st.op = tok.val
			}

		case tokString:
			// append raw string
			st := &stack[len(stack)-1]
			st.pieces[st.st] = addString(st.pieces[st.st], tok.val)
		}
	}

	// validate and return final state
	if len(stack) > 1 {
		return nil, errors.New("missing '}'")
	}
	if len(stack) == 0 {
		return nil, errors.New("fatal: expansion parse state empty")
	}

	result := stack[0].pieces[stLeft]
	if len(result) == 1 {
		return result[0], nil
	}
	return &splice{result}, nil
}

func cfgRoot(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	for {
		p := cfg.Parent()
		if p == nil {
			return cfg
		}

		cfg = p
	}
}

func addString(ps []varEvaler, s string) []varEvaler {
	if len(ps) == 0 {
		return []varEvaler{constExp(s)}
	}

	last := ps[len(ps)-1]
	c, ok := last.(constExp)
	if !ok {
		return append(ps, constExp(s))
	}

	ps[len(ps)-1] = constExp(string(c) + s)
	return ps
}

func (t tokenType) String() string {
	switch t {
	case tokOpen:
		return "<open>"
	case tokClose:
		return "<close>"
	case tokSep:
		return "<sep>"
	case tokString:
		return "<str>"
	}
	return "<unknown>"
}

func (t token) String() string {
	return fmt.Sprintf("(%v, %v)", t.typ, t.val)
}
