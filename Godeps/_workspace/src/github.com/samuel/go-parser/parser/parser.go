package parser

import (
	"errors"
	"io"
	"unicode"
)

type Position struct {
	Name   string
	Line   int
	Column int
	Offset int
}

type Input interface {
	Begin()
	End(rollback bool)

	Get(int) (string, error)
	Next() (rune, error)
	Pop(int)

	Position() Position
}

// Specifications for the parser
type Spec struct {
	CommentStart   string
	CommentEnd     string
	CommentLine    Parser
	NestedComments bool
	IdentStart     Parser
	IdentLetter    Parser
	ReservedNames  []string
}

type State struct {
	Spec  Spec
	Input Input
}

// A Parser is a function that takes a Input and returns any matches
// (Output) and whether or not the match was valid and any error
type Parser func(*State) (Output, bool, error)

// Output of Parsers
type Output interface{}

// Token that satisfies a condition.
func Satisfy(check func(c rune) bool) Parser {
	return func(st *State) (Output, bool, error) {
		target, err := st.Input.Next()
		if err == nil && check(target) {
			st.Input.Pop(1)
			return target, true, nil
		}

		return nil, false, err
	}
}

// Skip whitespace and comments
func Whitespace() Parser {
	return Skip(Many(Any(Satisfy(unicode.IsSpace), OneLineComment(), MultiLineComment())))
}

// func Comments() Parser {
// 	parser := Many(Any(Skip(Satisfy(unicode.IsSpace)), OneLineComment(), MultiLineComment()))
// 	return func(state *State) (Output, bool, error) {
// 		out, ok := parser(st)
// 		if !ok {
// 			return out, ok
// 		}
// 		// for _, o := range out.([]interface{}) {
// 		// 	switch s := o.(type) {
// 		// 	case []byte:
// 		// 		println(string(s))
// 		// 	case string:
// 		// 		println(s)
// 		// 	default:
// 		// 		panic(o)
// 		// 		panic("Unexpected type")
// 		// 	}
// 		// }
// 		return out, ok
// 	}
// }

func stringUntil(until rune) Parser {
	return func(st *State) (Output, bool, error) {
		out := make([]rune, 0)

		for {
			next, err := st.Input.Next()
			if err != nil {
				return string(out), false, err
			}
			st.Input.Pop(1)
			if next == until {
				break
			}
			out = append(out, next)
		}
		return string(out), true, nil
	}
}

func OneLineComment() Parser {
	return func(st *State) (Output, bool, error) {
		if st.Spec.CommentLine == nil {
			return nil, false, nil
		}

		return All(
			Try(st.Spec.CommentLine),
			stringUntil('\n'))(st)
	}
}

func MultiLineComment() Parser {
	return func(st *State) (Output, bool, error) {
		spec := st.Spec

		return All(
			String(spec.CommentStart),
			InComment())(st)
	}
}

func InComment() Parser {
	return func(st *State) (Output, bool, error) {
		if st.Spec.NestedComments {
			return inMulti()(st)
		}

		return inSingle()(st)
	}
}

func inMulti() Parser {
	return func(st *State) (Output, bool, error) {
		spec := st.Spec
		startEnd := spec.CommentStart + spec.CommentEnd

		return Any(
			Try(String(spec.CommentEnd)),
			All(MultiLineComment(), inMulti()),
			All(Many1(NoneOf(startEnd)), inMulti()),
			All(OneOf(startEnd), inMulti()))(st)
	}
}

func inSingle() Parser {
	return func(st *State) (Output, bool, error) {
		spec := st.Spec
		startEnd := spec.CommentStart + spec.CommentEnd

		return Any(
			Try(String(spec.CommentEnd)),
			All(Many1(NoneOf(startEnd)), inSingle()),
			All(OneOf(startEnd), inSingle()))(st)
	}
}

func OneOf(cs string) Parser {
	return func(st *State) (Output, bool, error) {
		next, err := st.Input.Next()
		if err != nil {
			return nil, false, err
		}

		for _, v := range cs {
			if v == next {
				st.Input.Pop(1)
				return v, true, nil
			}
		}

		return next, false, nil
	}
}

func NoneOf(cs string) Parser {
	return func(st *State) (Output, bool, error) {
		next, err := st.Input.Next()
		if err != nil {
			return nil, false, err
		}

		for _, v := range cs {
			if v == next {
				return v, false, nil
			}
		}

		st.Input.Pop(1)
		return next, true, nil
	}
}

func Skip(match Parser) Parser {
	return func(st *State) (Output, bool, error) {
		_, ok, err := match(st)
		return nil, ok, err
	}
}

func Token() Parser {
	return func(st *State) (Output, bool, error) {
		next, err := st.Input.Next()
		if err != nil {
			return next, false, err
		}
		st.Input.Pop(1)
		return next, true, nil
	}
}

// Match a parser and skip whitespace
func Lexeme(match Parser) Parser {
	return func(st *State) (Output, bool, error) {
		out, matched, err := match(st)
		if err != nil {
			return nil, false, err
		}

		if !matched {
			return nil, false, nil
		}

		Whitespace()(st)

		return out, true, nil
	}
}

// Match a parser 0 or more times.
func Many(match Parser) Parser {
	return func(st *State) (Output, bool, error) {
		matches := []interface{}{}
		for {
			out, parsed, err := match(st)
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, false, err
			} else if !parsed {
				break
			}

			if out != nil {
				matches = append(matches, out)
			}
		}

		return matches, true, nil
	}
}

// Match a parser 1 or more times.
func Many1(match Parser) Parser {
	return func(st *State) (Output, bool, error) {
		a, ok, err := match(st)
		if !ok || err != nil {
			return nil, false, err
		}

		rest, ok, err := Many(match)(st)
		if !ok || err != nil {
			return nil, false, err
		}

		as := rest.([]interface{})

		all := make([]interface{}, len(as)+1)
		all[0] = a
		for i := 0; i < len(as); i++ {
			all[i+1] = as[i]
		}

		return all, true, nil
	}
}

// Match a parser seperated by another parser 0 or more times.
// Trailing delimeters are valid.
func SepBy(delim Parser, match Parser) Parser {
	return func(st *State) (Output, bool, error) {
		matches := []interface{}{}
		for {
			out, parsed, err := match(st)
			if err != nil {
				return nil, false, err
			}

			if !parsed {
				break
			}

			matches = append(matches, out)

			_, sep, err := delim(st)
			if err != nil {
				return nil, false, err
			}
			if !sep {
				break
			}
		}

		return matches, true, nil
	}
}

// Go through the parsers until one matches.
func Any(parsers ...Parser) Parser {
	return func(st *State) (Output, bool, error) {
		for _, parser := range parsers {
			match, ok, err := parser(st)
			if ok || err != nil {
				return match, ok, err
			}
		}

		return nil, false, nil
	}
}

// Match all parsers, returning the final result. If one fails, it stops.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func All(parsers ...Parser) Parser {
	return func(st *State) (match Output, ok bool, err error) {
		for _, parser := range parsers {
			match, ok, err = parser(st)
			if !ok || err != nil {
				return
			}
		}

		return
	}
}

// Match all parsers, collecting their outputs into a vector.
// If one parser fails, the whole thing fails.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func Collect(parsers ...Parser) Parser {
	return func(st *State) (Output, bool, error) {
		matches := []interface{}{}
		for _, parser := range parsers {
			match, ok, err := parser(st)
			if !ok || err != nil {
				return nil, false, err
			}

			matches = append(matches, match)
		}

		return matches, true, nil
	}
}

// Try matching begin, match, and then end.
func Between(begin Parser, end Parser, match Parser) Parser {
	return func(st *State) (Output, bool, error) {
		parse, ok, err := Try(Collect(begin, match, end))(st)
		if !ok || err != nil {
			return nil, false, err
		}

		return parse.([]interface{})[1], true, nil
	}
}

// Lexeme parser for `match' wrapped in parens.
func Parens(match Parser) Parser { return Lexeme(Between(Symbol("("), Symbol(")"), match)) }

// Match a string and consume any following whitespace.
func Symbol(str string) Parser { return Lexeme(String(str)) }

// Match a string and pop the string's length from the input.
// NOTE: Consumes input on failure. Wrap calls in Try(...) to avoid.
func String(str string) Parser {
	return func(st *State) (Output, bool, error) {
		for _, v := range str {
			next, err := st.Input.Next()
			if err != nil || next != v {
				return nil, false, err
			}

			st.Input.Pop(1)
		}

		return str, true, nil
	}
}

// Try a parse and revert the state and position if it fails.
func Try(match Parser) Parser {
	return func(st *State) (Output, bool, error) {
		st.Input.Begin()
		out, ok, err := match(st)
		st.Input.End(!ok || err == io.EOF)
		if err == io.EOF {
			if ok {
				err = errors.New("parser: bad state: err == EOF but ok == true")
			} else {
				err = nil
			}
		}
		return out, ok, err
	}
}

func Ident() Parser {
	return func(st *State) (Output, bool, error) {
		sp := st.Spec
		n, ok, err := sp.IdentStart(st)
		if !ok || err != nil {
			return nil, ok, err
		}

		ns, ok, err := Many(sp.IdentLetter)(st)
		if !ok || err != nil {
			return nil, ok, err
		}

		rest := make([]rune, len(ns.([]interface{})))
		for k, v := range ns.([]interface{}) {
			rest[k] = v.(rune)
		}

		return string(n.(rune)) + string(rest), true, nil
	}
}

func Identifier() Parser {
	return Lexeme(Try(func(st *State) (Output, bool, error) {
		name, ok, err := Ident()(st)
		if !ok || err != nil {
			return name, ok, err
		}

		for _, v := range st.Spec.ReservedNames {
			if v == name {
				return nil, false, nil
			}
		}

		return name, true, nil
	}))
}
