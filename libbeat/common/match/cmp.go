package match

import "regexp/syntax"

// common predefined patterns
var (
	patDotStar          = mustParse(`.*`)
	patNullBeginDotStar = mustParse(`^.*`)
	patNullEndDotStar   = mustParse(`.*$`)

	patEmptyText      = mustParse(`^$`)
	patEmptyWhiteText = mustParse(`^\s*$`)

	// patterns matching any content
	patAny1 = patDotStar
	patAny2 = mustParse(`^.*`)
	patAny3 = mustParse(`^.*$`)
	patAny4 = mustParse(`.*$`)

	patBeginText = mustParse(`^`)
	patEndText   = mustParse(`$`)

	patDigits = mustParse(`\d`)
)

// isPrefixLiteral checks regular expression being literal checking string
// starting with literal pattern (like '^PATTERN')
func isPrefixLiteral(r *syntax.Regexp) bool {
	return r.Op == syntax.OpConcat &&
		len(r.Sub) == 2 &&
		r.Sub[0].Op == syntax.OpBeginText &&
		r.Sub[1].Op == syntax.OpLiteral
}

func isAltLiterals(r *syntax.Regexp) bool {
	if r.Op != syntax.OpAlternate {
		return false
	}

	for _, sub := range r.Sub {
		if sub.Op != syntax.OpLiteral {
			return false
		}
	}
	return true
}

func isExactLiteral(r *syntax.Regexp) bool {
	return r.Op == syntax.OpConcat &&
		len(r.Sub) == 3 &&
		r.Sub[0].Op == syntax.OpBeginText &&
		r.Sub[1].Op == syntax.OpLiteral &&
		r.Sub[2].Op == syntax.OpEndText
}

func isOneOfLiterals(r *syntax.Regexp) bool {
	return r.Op == syntax.OpConcat &&
		len(r.Sub) == 3 &&
		r.Sub[0].Op == syntax.OpBeginText &&
		isAltLiterals(r.Sub[1]) &&
		r.Sub[2].Op == syntax.OpEndText
}

// isPrefixAltLiterals checks regular expression being alternative literals
// starting with literal pattern (like '^PATTERN')
func isPrefixAltLiterals(r *syntax.Regexp) bool {
	isPrefixAlt := r.Op == syntax.OpConcat &&
		len(r.Sub) == 2 &&
		r.Sub[0].Op == syntax.OpBeginText &&
		r.Sub[1].Op == syntax.OpAlternate
	if !isPrefixAlt {
		return false
	}

	for _, sub := range r.Sub[1].Sub {
		if sub.Op != syntax.OpLiteral {
			return false
		}
	}
	return true
}

func isPrefixNumDate(r *syntax.Regexp) bool {
	if r.Op != syntax.OpConcat || r.Sub[0].Op != syntax.OpBeginText {
		return false
	}

	i := 1
	if r.Sub[i].Op == syntax.OpLiteral {
		i++
	}

	// check starts with digits `\d{n}` or `[0-9]{n}`
	if !isMultiDigits(r.Sub[i]) {
		return false
	}
	i++

	for i < len(r.Sub) {
		// check separator
		if r.Sub[i].Op != syntax.OpLiteral {
			return false
		}
		i++

		// regex has 'OpLiteral' suffix, without any more digits/patterns following
		if i == len(r.Sub) {
			return true
		}

		// check digits
		if !isMultiDigits(r.Sub[i]) {
			return false
		}
		i++
	}

	return true
}

// isdotStar checks the term being `.*`.
func isdotStar(r *syntax.Regexp) bool {
	return eqRegex(r, patDotStar)
}

func isEmptyText(r *syntax.Regexp) bool {
	return eqRegex(r, patEmptyText)
}

func isEmptyTextWithWhitespace(r *syntax.Regexp) bool {
	return eqRegex(r, patEmptyWhiteText)
}

func isAnyMatch(r *syntax.Regexp) bool {
	return eqRegex(r, patAny1) ||
		eqRegex(r, patAny2) ||
		eqRegex(r, patAny3) ||
		eqRegex(r, patAny4)
}

func isDigitMatch(r *syntax.Regexp) bool {
	return eqRegex(r, patDigits)
}

func isMultiDigits(r *syntax.Regexp) bool {
	return isConcatRepetition(r) && isDigitMatch(r.Sub[0])
}

func isConcatRepetition(r *syntax.Regexp) bool {
	if r.Op != syntax.OpConcat {
		return false
	}

	first := r.Sub[0]
	for _, other := range r.Sub {
		if other != first { // concat repetitions reuse references => compare pointers
			return false
		}
	}

	return true
}

func eqRegex(r, proto *syntax.Regexp) bool {
	unmatchable := r.Op != proto.Op || r.Flags != proto.Flags ||
		(r.Min != proto.Min) || (r.Max != proto.Max) ||
		(len(r.Sub) != len(proto.Sub)) ||
		(len(r.Rune) != len(proto.Rune))

	if unmatchable {
		return false
	}

	for i := range r.Sub {
		if !eqRegex(r.Sub[i], proto.Sub[i]) {
			return false
		}
	}

	for i := range r.Rune {
		if r.Rune[i] != proto.Rune[i] {
			return false
		}
	}
	return true
}

func eqPrefixAnyRegex(r *syntax.Regexp, protos ...*syntax.Regexp) bool {
	for _, proto := range protos {
		if eqPrefixRegex(r, proto) {
			return true
		}
	}
	return false
}

func eqPrefixRegex(r, proto *syntax.Regexp) bool {
	if r.Op != syntax.OpConcat {
		return false
	}

	if proto.Op != syntax.OpConcat {
		if len(r.Sub) == 0 {
			return false
		}
		return eqRegex(r.Sub[0], proto)
	}

	if len(r.Sub) < len(proto.Sub) {
		return false
	}

	for i := range proto.Sub {
		if !eqRegex(r.Sub[i], proto.Sub[i]) {
			return false
		}
	}
	return true
}

func eqSuffixAnyRegex(r *syntax.Regexp, protos ...*syntax.Regexp) bool {
	for _, proto := range protos {
		if eqSuffixRegex(r, proto) {
			return true
		}
	}
	return false
}

func eqSuffixRegex(r, proto *syntax.Regexp) bool {
	if r.Op != syntax.OpConcat {
		return false
	}

	if proto.Op != syntax.OpConcat {
		i := len(r.Sub) - 1
		if i < 0 {
			return false
		}
		return eqRegex(r.Sub[i], proto)
	}

	if len(r.Sub) < len(proto.Sub) {
		return false
	}

	d := len(r.Sub) - len(proto.Sub)
	for i := range proto.Sub {
		if !eqRegex(r.Sub[d+i], proto.Sub[i]) {
			return false
		}
	}
	return true
}

func mustParse(pattern string) *syntax.Regexp {
	r, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		panic(err)
	}
	return r
}
