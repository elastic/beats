package http

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/libbeat/common/match"
)

func TestCheckBody(t *testing.T) {

	var matchTests = []struct {
		description string
		body        string
		patterns    []match.Matcher
		result      bool
	}{
		{
			"Single regex that matches",
			"ok",
			[]match.Matcher{match.MustCompile("ok")},
			true,
		},
		{
			"Regex matching json example",
			`{"status": "ok"}`,
			[]match.Matcher{match.MustCompile(`{"status": "ok"}`)},
			true,
		},
		{
			"Regex matching first line of multiline body string",
			`first line
			second line`,
			[]match.Matcher{match.MustCompile("first")},
			true,
		},
		{
			"Regex matching lastline of multiline body string",
			`first line
			second line`,
			[]match.Matcher{match.MustCompile("second")},
			true,
		},
		{
			"Regex matching multiple lines of multiline body string",
			`first line
			second line
			third line`,
			[]match.Matcher{match.MustCompile("(?s)first.*second.*third")},
			true,
		},
		{
			"Regex not matching multiple lines of multiline body string",
			`first line
			second line
			third line`,
			[]match.Matcher{match.MustCompile("(?s)first.*fourth.*third")},
			false,
		},
		{
			"Single regex that doesn't match",
			"ok",
			[]match.Matcher{match.MustCompile("notok")},
			false,
		},
		{
			"Multiple regex match where at least one must match",
			"ok",
			[]match.Matcher{match.MustCompile("ok"), match.MustCompile("yay")},
			true,
		},
		{
			"Multiple regex match where none of the patterns match",
			"ok",
			[]match.Matcher{match.MustCompile("notok"), match.MustCompile("yay")},
			false,
		},
	}

	for _, test := range matchTests {
		t.Run(test.description, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, test.body)
			}))
			defer ts.Close()

			res, err := http.Get(ts.URL)
			if err != nil {
				log.Fatal(err)
			}

			check := checkBody(test.patterns)(res)

			if result := (check == nil); result != test.result {
				if test.result {
					t.Fatalf("Expected at least one of patterns: %s to match body: %s", test.patterns, test.body)
				} else {
					t.Fatalf("Did not expect patterns: %s to match body: %s", test.patterns, test.body)
				}
			}
		})
	}
}
