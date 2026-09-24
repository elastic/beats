// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package match

import (
	"bytes"
	"fmt"
	"regexp"
	"testing"
)

type testContent struct {
	name  string
	lines [][]byte
}

type benchRunner struct {
	title string
	f     func(*testing.B)
}

var allContents = []testContent{
	mixedContent,
	logContent,
	logContent2,
	logContentLevel,
}

var mixedContent = makeContent("mixed", `Lorem ipsum dolor sit amet,
PATTERN consectetur adipiscing elit. Nam vitae turpis augue.
 Quisque euismod erat tortor, posuere auctor elit fermentum vel. Proin in odio

23-08-2016 eleifend, maximus turpis non, lacinia ligula. Nullam vel pharetra quam, id egestas
   
massa. Sed a vestibulum libero. Sed tellus lorem, imperdiet non nisl ac,
 aliquet placerat magna. Sed PATTERN in bibendum eros. Curabitur ut pretium neque. Sed
23-08-2016 egestas elit et leo consectetur, nec dignissim arcu ultricies. Sed molestie tempor

erat, a maximus sapien rutrum ut. Curabitur congue condimentum dignissim.
 Mauris hendrerit, velit nec accumsan egestas, augue justo tincidunt risus,
  
a facilisis nulla augue PATTERN eu metus. Duis vel neque sit amet nunc elementum viverra
eu ut ligula. Mauris et libero lacus.`)

var logContent = makeContent("simple_log", `23-08-2016 15:10:01 - Lorem ipsum dolor sit amet,
23-08-2016 15:10:02 - PATTERN consectetur adipiscing elit. Nam vitae turpis augue.
23-08-2016 15:10:03 -  Quisque euismod erat tortor, posuere auctor elit fermentum vel. Proin in odio
23-08-2016 15:10:05 - 23-08-2016 eleifend, maximus turpis non, lacinia ligula. Nullam vel pharetra quam, id egestas
23-08-2016 15:10:07 - massa. Sed a vestibulum libero. Sed tellus lorem, imperdiet non nisl ac,
23-08-2016 15:10:08 -  aliquet placerat magna. Sed PATTERN in bibendum eros. Curabitur ut pretium neque. Sed
23-08-2016 15:10:09 - 23-08-2016 egestas elit et leo consectetur, nec dignissim arcu ultricies. Sed molestie tempor
23-08-2016 15:10:11 - erat, a maximus sapien rutrum ut. Curabitur congue condimentum dignissim.
23-08-2016 15:10:12 -  Mauris hendrerit, velit nec accumsan egestas, augue justo tincidunt risus,
23-08-2016 15:10:14 - a facilisis nulla augue PATTERN eu metus. Duis vel neque sit amet nunc elementum viverra
eu ut ligula. Mauris et libero lacus.
`)

var logContent2 = makeContent("simple_log2", `2016-08-23 15:10:01 - DEBUG - Lorem ipsum dolor sit amet,
2016-08-23 15:10:02 - INFO - PATTERN consectetur adipiscing elit. Nam vitae turpis augue.
2016-08-23 15:10:03 -  DEBUG - Quisque euismod erat tortor, posuere auctor elit fermentum vel. Proin in odio
2016-08-23 15:10:05 - ERROR - 23-08-2016 eleifend, maximus turpis non, lacinia ligula. Nullam vel pharetra quam, id egestas
2016-08-23 15:10:07 - WARN - massa. Sed a vestibulum libero. Sed tellus lorem, imperdiet non nisl ac,
2016-08-23 15:10:08 - CRIT - aliquet placerat magna. Sed PATTERN in bibendum eros. Curabitur ut pretium neque. Sed
2016-08-23 15:10:09 - DEBUG - 23-08-2016 egestas elit et leo consectetur, nec dignissim arcu ultricies. Sed molestie tempor
2016-08-23 15:10:11 - ERROR - erat, a maximus sapien rutrum ut. Curabitur congue condimentum dignissim.
2016-08-23 15:10:12 - INFO - Mauris hendrerit, velit nec accumsan egestas, augue justo tincidunt risus,
2016-08-23 15:10:14 - INFO - a facilisis nulla augue PATTERN eu metus. Duis vel neque sit amet nunc elementum viverra eu ut ligula. Mauris et libero lacus.
`)

var logContentLevel = makeContent("simple_log_with_level", `DEBUG - 2016-08-23 15:10:01 - Lorem ipsum dolor sit amet,
INFO - 2016-08-23 15:10:02 - PATTERN consectetur adipiscing elit. Nam vitae turpis augue.
DEBUG - 2016-08-23 15:10:03 -  Quisque euismod erat tortor, posuere auctor elit fermentum vel. Proin in odio
ERROR - 2016-08-23 15:10:05 - 23-08-2016 eleifend, maximus turpis non, lacinia ligula. Nullam vel pharetra quam, id egestas
WARN - 2016-08-23 15:10:07 - massa. Sed a vestibulum libero. Sed tellus lorem, imperdiet non nisl ac,
CRIT - 2016-08-23 15:10:08 -  aliquet placerat magna. Sed PATTERN in bibendum eros. Curabitur ut pretium neque. Sed
DEBUG - 2016-08-23 15:10:09 - 23-08-2016 egestas elit et leo consectetur, nec dignissim arcu ultricies. Sed molestie tempor
ERROR -2016-08-23 15:10:11 - erat, a maximus sapien rutrum ut. Curabitur congue condimentum dignissim.
DEBUG - 2016-08-23 15:10:12 -  Mauris hendrerit, velit nec accumsan egestas, augue justo tincidunt risus,
INFO - 2016-08-23 15:10:14 - a facilisis nulla augue PATTERN eu metus. Duis vel neque sit amet nunc elementum viverra
eu ut ligula. Mauris et libero lacus.
`)

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
		{"startsWithDate2", `^\d{4}-\d{2}-\d{2}`},
		{"startsWithDate3", `^\d\d\d\d-\d\d-\d\d`},
		{"startsWithDate4", `^20\d{2}-\d{2}-\d{2}`},
		{"startsWithDateAndSpace", `^\d{4}-\d{2}-\d{2} `},
		{"startsWithLevel", `^(DEBUG|INFO|WARN|ERR|CRIT)`},
		{"hasLevel", `(DEBUG|INFO|WARN|ERR|CRIT)`},
		{"contains 'PATTERN'", `PATTERN`},
		{"contains 'PATTERN' with '.*", `.*PATTERN.*`},
		{"empty line", `^$`},
		{"empty line with optional whitespace", `^\s*$`},
	}

	runTitle := func(matcher, name, content string) string {
		return fmt.Sprintf("Name=%v, Matcher=%v, Content=%v", name, matcher, content)
	}

	for i, pattern := range patterns {
		b.Logf("benchmark (%v): %v", i, pattern.title)

		regex := regexp.MustCompile(pattern.regex)
		matcher := MustCompile(pattern.regex)

		b.Logf("regex: %v", regex)
		b.Logf("matcher: %v", matcher)

		for _, content := range allContents {
			title := runTitle("Regex", pattern.title, content.name)
			runner := makeRunner(title, content.lines, regex.Match)
			b.Run(runner.title, runner.f)

			title = runTitle("Match", pattern.title, content.name)
			runner = makeRunner(title, content.lines, matcher.Match)
			b.Run(runner.title, runner.f)
		}
	}
}

func makeRunner(title string, content [][]byte, m func([]byte) bool) benchRunner {
	return benchRunner{
		title,
		func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for _, line := range content {
					m(line)
				}
			}
		},
	}
}

func makeContent(name, s string) testContent {
	return testContent{
		name,
		bytes.Split([]byte(s), []byte("\n")),
	}
}
