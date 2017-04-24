package match

import (
	"regexp"
	"regexp/syntax"
)

func compile(r *syntax.Regexp) (stringMatcher, error) {
	switch {
	case r.Op == syntax.OpLiteral:
		s := string(r.Rune)
		return &substringMatcher{s, []byte(s)}, nil

	case isExactLiteral(r):
		s := string(r.Sub[1].Rune)
		return &equalsMatcher{s, []byte(s)}, nil

	case isAltLiterals(r):
		var literals [][]byte
		for _, sub := range r.Sub {
			literals = append(literals, []byte(string(sub.Rune)))
		}
		return &altSubstringMatcher{literals}, nil

	case isOneOfLiterals(r):
		var literals [][]byte
		for _, sub := range r.Sub[1].Sub {
			literals = append(literals, []byte(string(sub.Rune)))
		}
		return &oneOfMatcher{literals}, nil

	case isPrefixLiteral(r):
		s := []byte(string(r.Sub[1].Rune))
		return &prefixMatcher{s}, nil

	case isPrefixAltLiterals(r):
		var literals [][]byte
		for _, sub := range r.Sub[1].Sub {
			literals = append(literals, []byte(string(sub.Rune)))
		}
		return &altPrefixMatcher{literals}, nil

	case isPrefixNumDate(r):
		return compilePrefixNumDate(r)

	case isEmptyText(r):
		var m *emptyStringMatcher
		return m, nil

	case isEmptyTextWithWhitespace(r):
		var m *emptyWhiteStringMatcher
		return m, nil

	case isAnyMatch(r):
		var m *matchAny
		return m, nil

	default:

		r, err := regexp.Compile(r.String())
		if err != nil {
			return nil, err
		}
		return r, nil
	}
}

func compilePrefixNumDate(r *syntax.Regexp) (stringMatcher, error) {
	m := &prefixNumDate{}

	i := 1
	if r.Sub[i].Op == syntax.OpLiteral {
		m.prefix = []byte(string(r.Sub[i].Rune))
		i++
	}

	digitLen := func(r *syntax.Regexp) int {
		if r.Op == syntax.OpConcat {
			return len(r.Sub)
		}
		return 1
	}

	var digits []int
	var seps [][]byte

	digits = append(digits, digitLen(r.Sub[i]))
	i++

	for i < len(r.Sub) {
		lit := []byte(string(r.Sub[i].Rune))
		i++

		// capture literal suffix
		if i == len(r.Sub) {
			m.suffix = lit
			break
		}

		seps = append(seps, lit)
		digits = append(digits, digitLen(r.Sub[i]))
		i++
	}

	minLen := len(m.prefix) + len(m.suffix)
	for _, d := range digits {
		minLen += d
	}
	for _, sep := range seps {
		minLen += len(sep)
	}

	m.digits = digits
	m.seps = seps
	m.minLen = minLen

	return m, nil
}
