package match

import "regexp/syntax"

type Matcher struct {
	stringMatcher
}

type ExactMatcher struct {
	stringMatcher
}

type stringMatcher interface {
	// MatchString tries to find a matching substring.
	MatchString(s string) (matched bool)

	// Match tries to find a matching substring.
	Match(bs []byte) (matched bool)

	// Describe the generator
	String() string
}

func MustCompile(pattern string) Matcher {
	m, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return m
}

func MustCompileExact(pattern string) ExactMatcher {
	m, err := CompileExact(pattern)
	if err != nil {
		panic(err)
	}
	return m
}

// CompileString matches a substring only, the input is not interpreted as
// regular expression
func CompileString(in string) (Matcher, error) {
	if in == "" {
		return Matcher{(*emptyStringMatcher)(nil)}, nil
	}
	return Matcher{&substringMatcher{in, []byte(in)}}, nil
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
	m, err := compile(regex)
	return Matcher{m}, err
}

func CompileExact(pattern string) (ExactMatcher, error) {
	regex, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return ExactMatcher{}, err
	}

	regex = regex.Simplify()
	if regex.Op != syntax.OpConcat {
		regex = &syntax.Regexp{
			Op: syntax.OpConcat,
			Sub: []*syntax.Regexp{
				patBeginText,
				regex,
				patEndText,
			},
			Flags: regex.Flags,
		}
	} else {
		if !eqPrefixRegex(regex, patBeginText) {
			regex.Sub = append([]*syntax.Regexp{patBeginText}, regex.Sub...)
		}
		if !eqSuffixRegex(regex, patEndText) {
			regex.Sub = append(regex.Sub, patEndText)
		}
	}

	regex = optimize(regex).Simplify()
	m, err := compile(regex)
	return ExactMatcher{m}, err
}

func (m *Matcher) Unpack(s string) error {
	tmp, err := Compile(s)
	if err != nil {
		return err
	}

	*m = tmp
	return nil
}

func (m *Matcher) MatchAnyString(strs []string) bool {
	return matchAnyStrings(m.stringMatcher, strs)
}

func (m *Matcher) MatchAllStrings(strs []string) bool {
	return matchAllStrings(m.stringMatcher, strs)
}

func (m *ExactMatcher) MatchAnyString(strs []string) bool {
	return matchAnyStrings(m.stringMatcher, strs)
}

func (m *ExactMatcher) MatchAllStrings(strs []string) bool {
	return matchAllStrings(m.stringMatcher, strs)
}

func (m *ExactMatcher) Unpack(s string) error {
	tmp, err := CompileExact(s)
	if err != nil {
		return err
	}

	*m = tmp
	return nil
}

func matchAnyStrings(m stringMatcher, strs []string) bool {
	for _, s := range strs {
		if m.MatchString(s) {
			return true
		}
	}
	return false
}

func matchAllStrings(m stringMatcher, strs []string) bool {
	for _, s := range strs {
		if !m.MatchString(s) {
			return false
		}
	}
	return true
}
