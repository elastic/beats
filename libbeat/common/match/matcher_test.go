package match

import (
	"regexp"
	"strings"
	"testing"
)

type testMatcher interface {
	// Match([]byte) bool
	// MatchReader(io.RuneReader) bool
	MatchString(string) bool
}

type matcherFactory func(pattern string) (testMatcher, error)

func BenchmarkBeginning(b *testing.B) {
	pattern := `^PATTERN`
	b.Run("Matcher=Regex", makeRegexRunner(pattern))
	b.Run("Matcher=Match", makeMatchRunner(pattern))
}

func BenchmarkBeginningSpace(b *testing.B) {
	pattern := `^ `
	b.Run("Matcher=Regex", makeRegexRunner(pattern))
	b.Run("Matcher=Match", makeMatchRunner(pattern))
}

func BenchmarkBeginningDate(b *testing.B) {
	pattern := `^\d{2}-\d{2}-\d{4}`
	b.Run("Matcher=Regex", makeRegexRunner(pattern))
	b.Run("Matcher=Match", makeMatchRunner(pattern))
}

func BenchmarkStringPatternRegex(b *testing.B) {
	pattern := `PATTERN`
	b.Run("Matcher=Regex", makeRegexRunner(pattern))
	b.Run("Matcher=Match", makeMatchRunner(pattern))
}

func BenchmarkStringPatternDotStarRegex(b *testing.B) {
	pattern := `.*PATTERN.*`
	b.Run("Matcher=Regex", makeRegexRunner(pattern))
	b.Run("Matcher=Match", makeMatchRunner(pattern))
}

func runBenchStrings(
	b *testing.B,
	factory matcherFactory,
	pattern string,
	content []string,
) bool {
	matcher, err := factory(pattern)
	if err != nil {
		b.Fatal(err)
	}

	found := false
	for i := 0; i < b.N; i++ {
		for _, line := range content {
			b := matcher.MatchString(line)
			found = found || b
		}
	}

	return found
}

func regexFactory(pattern string) (testMatcher, error) {
	return regexp.Compile(pattern)
}

func matchFactory(pattern string) (testMatcher, error) {
	return Compile(pattern)
}

var commonContent = strings.Split(`Lorem ipsum dolor sit amet,
PATTERN consectetur adipiscing elit. Nam vitae turpis augue.
 Quisque euismod erat tortor, posuere auctor elit fermentum vel. Proin in odio
23-08-2016 eleifend, maximus turpis non, lacinia ligula. Nullam vel pharetra quam, id egestas
massa. Sed a vestibulum libero. Sed tellus lorem, imperdiet non nisl ac,
 aliquet placerat magna. Sed PATTERN in bibendum eros. Curabitur ut pretium neque. Sed
23-08-2016 egestas elit et leo consectetur, nec dignissim arcu ultricies. Sed molestie tempor
erat, a maximus sapien rutrum ut. Curabitur congue condimentum dignissim.
 Mauris hendrerit, velit nec accumsan egestas, augue justo tincidunt risus,
a facilisis nulla augue PATTERN eu metus. Duis vel neque sit amet nunc elementum viverra
eu ut ligula. Mauris et libero lacus.`, "\n")

func makeRunner(m func(string) bool) func(*testing.B) {
	return func(b *testing.B) {
		found := false
		for i := 0; i < b.N; i++ {
			for _, line := range commonContent {
				b := m(line)
				found = found || b
			}
		}
		if found == false {
			b.Error("no matches found")
		}
	}
}

func makeRegexRunner(pattern string) func(*testing.B) {
	return makeRunner(regexp.MustCompile(pattern).MatchString)
}

func makeMatchRunner(pattern string) func(*testing.B) {
	return makeRunner(MustCompile(pattern).MatchString)
}
