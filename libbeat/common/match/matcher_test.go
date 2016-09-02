package match

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

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

func BenchmarkPatterns(b *testing.B) {
	patterns := []struct {
		title string
		regex string
	}{
		{"match any 1", `^.*$`},
		{"match any 2", `.*`},
		{"startsWith 'PATTERN'", `^PATTERN`},
		{"startsWith ' '", `^ `},
		{"startsWithDate", `^\d{2}-\d{2}-\d{4}`},
		{"contains 'PATTERN'", `PATTERN`},
		{"contains 'PATTERN' with '.*", `.*PATTERN.*`},
		{"empty line", `^$`},
		{"empty line with optional whitespace", `^\s*$`},
	}

	runTitle := func(matcher, name string) string {
		return fmt.Sprintf("Name=%v, Matcher=%v", name, matcher)
	}

	for i, pattern := range patterns {
		b.Logf("benchmark (%v): %v", i, pattern.title)

		regex := makeRunner(regexp.MustCompile(pattern.regex).MatchString)
		matcher := makeRunner(MustCompile(pattern.regex).MatchString)

		b.Run(runTitle("Regex", pattern.title), regex)
		b.Run(runTitle("Match", pattern.title), matcher)
	}
}

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
