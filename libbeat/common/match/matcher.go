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

func (m *ExactMatcher) Unpack(s string) error {
	tmp, err := CompileExact(s)
	if err != nil {
		return err
	}

	*m = tmp
	return nil
}
