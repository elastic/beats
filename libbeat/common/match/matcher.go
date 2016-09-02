package match

import (
	"bytes"
	"errors"
	"regexp"
	"regexp/syntax"
	"strings"
	"unicode/utf8"
)

type Matcher struct {
	stringMatcher
}

type stringMatcher interface {
	// matchString tries to find a matching substring. If matched is false,
	// the matcher didn't match. If matched is true, rest contains the yet
	// unmatched substring which can be used for further matching (e.g. when
	// concatenating matches)
	MatchString(s string) (matched bool)
	Match(bs []byte) (matched bool)
}

type substringMatcher struct {
	s  string
	bs []byte
}

type prefixMatcher struct {
	s []byte
}

type emptyStringMatcher struct{}

type emptyWhiteStringMatcher struct{}

type matchAny struct{}

// common predefined patterns
var (
	patDotStar        = mustParse(`.*`)
	patEmptyText      = mustParse(`^$`)
	patEmptyWhiteText = mustParse(`^\s*$`)

	// patterns matching any content
	patAny1 = patDotStar
	patAny2 = mustParse(`^.*`)
	patAny3 = mustParse(`^.*$`)
	patAny4 = mustParse(`.*$`)
)

func MustCompile(pattern string) Matcher {
	m, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return m
}

// Compile regular expression to string matcher. String matcher by default uses
// regular expressions as provided by regexp library, but tries to optimize some
// common cases, replacing expensive patterns with cheaper custom implementations
// or removing terms not necessary for string matching.
func Compile(pattern string) (Matcher, error) {
	regex, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return Matcher{}, err
	}

	regex = optimize(regex).Simplify()
	return compile(regex)
}

func (m *Matcher) Unpack(v interface{}) error {
	s, ok := v.(string)
	if !ok {
		return errors.New("requires regular expression")
	}

	tmp, err := Compile(s)
	if err != nil {
		return err
	}

	*m = tmp
	return nil
}

func compile(r *syntax.Regexp) (Matcher, error) {
	switch {
	case r.Op == syntax.OpLiteral:
		s := string(r.Rune)
		return Matcher{&substringMatcher{s, []byte(s)}}, nil

	case isPrefixLiteral(r):
		s := []byte(string(r.Sub[0].Rune))
		return Matcher{&prefixMatcher{s}}, nil

	case isEmptyText(r):
		var m *emptyStringMatcher
		return Matcher{m}, nil

	case isEmptyTextWithWhitespace(r):
		var m *emptyWhiteStringMatcher
		return Matcher{m}, nil

	case isAnyMatch(r):
		var m *matchAny
		return Matcher{m}, nil

	default:
		r, err := regexp.Compile(r.String())
		if err != nil {
			return Matcher{}, err
		}
		return Matcher{r}, nil
	}
}

func (m *substringMatcher) MatchString(s string) bool {
	return strings.Contains(s, m.s)
}

func (m *substringMatcher) Match(bs []byte) bool {
	return bytes.Contains(bs, m.bs)
}

func (m *prefixMatcher) MatchString(s string) bool {
	return len(s) >= len(m.s) && s[0:len(m.s)] == string(m.s)
}

func (m *prefixMatcher) Match(bs []byte) bool {
	return len(bs) >= len(m.s) && bytes.Equal(bs[0:len(m.s)], m.s)
}

func (m *emptyStringMatcher) MatchString(s string) bool {
	return len(s) == 0
}

func (m *emptyStringMatcher) Match(bs []byte) bool {
	return len(bs) == 0
}

func (m *emptyWhiteStringMatcher) MatchString(s string) bool {
	for _, r := range s {
		if !(r == ' ' || ('\t' <= r && r <= '\n') || ('\f' <= r && r <= 'r')) {
			return false
		}
	}
	return true
}

func (m *emptyWhiteStringMatcher) Match(bs []byte) bool {
	for i := 0; i < len(bs); {
		r, size := utf8.DecodeRune(bs[i:])
		i += size
		if !(r == ' ' || ('\t' <= r && r <= '\n') || ('\f' <= r && r <= 'r')) {
			return false
		}
	}
	return true
}

func (m *matchAny) Match(_ []byte) bool       { return true }
func (m *matchAny) MatchString(_ string) bool { return true }

type trans func(*syntax.Regexp) (bool, *syntax.Regexp)

var transformations = []trans{
	simplify,
	uncapture,
	trimLeft,
	trimRight,
	unconcat,
}

// optimize runs minimal regular expression optimizations
// until fix-point.
func optimize(r *syntax.Regexp) *syntax.Regexp {
	for {
		changed := false
		for _, t := range transformations {
			var upd bool
			upd, r = t(r)
			changed = changed || upd
		}

		if changed == false {
			return r
		}
	}
}

// Simplify regular expression by stdlib.
func simplify(r *syntax.Regexp) (bool, *syntax.Regexp) {
	return false, r.Simplify()
}

// uncapture optimizes regular expression by removing capture groups from
// regular expression potentially allocating memory when executed.
func uncapture(r *syntax.Regexp) (bool, *syntax.Regexp) {
	if r.Op == syntax.OpCapture {
		// try to uncapture
		if len(r.Sub) == 1 {
			_, sub := uncapture(r.Sub[0])
			return true, sub
		}

		tmp := *r
		tmp.Op = syntax.OpConcat
		r = &tmp
	}

	sub := make([]*syntax.Regexp, len(r.Sub))
	modified := false
	for i := range r.Sub {
		var m bool
		m, sub[i] = uncapture(r.Sub[i])
		modified = modified || m
	}

	if !modified {
		return false, r
	}

	tmp := *r
	tmp.Sub = sub
	return true, &tmp
}

// trimLeft removes not required '.*' from beginning of regular expressions.
func trimLeft(r *syntax.Regexp) (bool, *syntax.Regexp) {
	if eqPrefixRegex(r, patDotStar) {
		tmp := *r
		tmp.Sub = tmp.Sub[1:]
		return true, &tmp
	}
	return false, r
}

// trimLeft removes not required '.*' from end of regular expressions.
func trimRight(r *syntax.Regexp) (bool, *syntax.Regexp) {
	if eqSuffixRegex(r, patDotStar) {
		i := len(r.Sub) - 1
		tmp := *r
		tmp.Sub = tmp.Sub[0:i]
		return true, &tmp
	}
	return false, r
}

// unconcat removes intermediate regular expression concatenations generated by
// parser if concatenation contains only 1 element. Removal of object from
// parse-tree can enable other optimization to fire.
func unconcat(r *syntax.Regexp) (bool, *syntax.Regexp) {
	if r.Op != syntax.OpConcat || len(r.Sub) > 1 {
		return false, r
	}

	if len(r.Sub) == 0 {
		return true, &syntax.Regexp{
			Op:    syntax.OpEmptyMatch,
			Flags: r.Flags,
		}
	}

	if len(r.Sub) == 1 {
		return true, r.Sub[0]
	}

	return false, r
}

// isPrefixLiteral checks regular expression being literal checking string
// starting with literal pattern (like '^PATTERN')
func isPrefixLiteral(r *syntax.Regexp) bool {
	return r.Op == syntax.OpConcat &&
		len(r.Sub) == 2 &&
		r.Sub[0].Op == syntax.OpBeginText &&
		r.Sub[1].Op == syntax.OpLiteral
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
