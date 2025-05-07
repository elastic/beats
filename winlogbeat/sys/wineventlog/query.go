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

package wineventlog

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
)

const (
	query = `<QueryList>
  <Query Id="0">
{{- if .Select}}{{range $s := .Select}}
    <Select Path="{{$.Path}}">*[System[{{join $s " and "}}]]</Select>{{end}}
{{- else}}
    <Select Path="{{.Path}}">*</Select>
{{- end}}
{{- if .Suppress}}
    <Suppress Path="{{.Path}}">*[System[{{.Suppress}}]]</Suppress>{{end}}
  </Query>
</QueryList>`

	queryClauseLimit = 21
)

var (
	templateFuncMap      = template.FuncMap{"join": strings.Join}
	queryTemplate        = template.Must(template.New("query").Funcs(templateFuncMap).Parse(query))
	incEventIDRegex      = regexp.MustCompile(`^\d+$`)
	incEventIDRangeRegex = regexp.MustCompile(`^(\d+)\s*-\s*(\d+)$`)
	excEventIDRegex      = regexp.MustCompile(`^-(\d+)$`)
	excEventIDRangeRegex = regexp.MustCompile(`^-(\d+)\s*-\s*(\d+)$`)
)

// Query that identifies the source of the events and one or more selectors or
// suppressors.
type Query struct {
	// Name of the channel or the URI path to the log file that contains the
	// events to query. The path to files must be a URI like file://C:/log.evtx.
	Log string

	IgnoreOlder time.Duration // Ignore records older than this time period.

	// Whitelist and blacklist of event IDs. The value is a comma-separated
	// list. The accepted values are single event IDs to include (e.g. 4634), a
	// range of event IDs to include (e.g. 4400-4500), and single event IDs to
	// exclude (e.g. -4410).
	EventID string

	// Level or levels to include. The value is a comma-separated list of levels
	// to include. The accepted levels are verbose (5), information (4),
	// warning (3), error (2), and critical (1).
	Level string

	// Providers (sources) to include records from.
	Provider []string
}

// Build builds a query from the given parameters. The query is returned as a
// XML string and can be used with Subscribe function.
func (q Query) Build() (string, error) {
	qp, err := newQueryParams(q)
	if err != nil {
		return "", err
	}
	return executeTemplate(queryTemplate, qp)
}

// queryParams are the parameters that are used to create a query from a
// template.
type queryParams struct {
	ignoreOlder        string
	level              string
	provider           string
	selectEventFilters []string

	Path     string
	Select   [][]string
	Suppress string
}

func newQueryParams(q Query) (*queryParams, error) {
	var errs []error
	if q.Log == "" {
		errs = append(errs, fmt.Errorf("empty log name"))
	}
	qp := &queryParams{
		Path: q.Log,
	}
	qp.withIgnoreOlder(q)
	qp.withProvider(q)
	if err := qp.withEventFilters(q); err != nil {
		errs = append(errs, err)
	}
	if err := qp.withLevel(q); err != nil {
		errs = append(errs, err)
	}
	qp.buildSelects()
	return qp, errors.Join(errs...)
}

func (qp *queryParams) withIgnoreOlder(q Query) {
	if q.IgnoreOlder <= 0 {
		return
	}
	ms := q.IgnoreOlder.Nanoseconds() / int64(time.Millisecond)
	qp.ignoreOlder = fmt.Sprintf("TimeCreated[timediff(@SystemTime) &lt;= %d]", ms)
}

func (qp *queryParams) withEventFilters(q Query) error {
	if q.EventID == "" {
		return nil
	}

	var includes []string
	var excludes []string
	components := strings.Split(q.EventID, ",")
	for _, c := range components {
		c = strings.TrimSpace(c)
		switch {
		case incEventIDRegex.MatchString(c):
			includes = append(includes, fmt.Sprintf("EventID=%s", c))
		case excEventIDRegex.MatchString(c):
			m := excEventIDRegex.FindStringSubmatch(c)
			excludes = append(excludes, fmt.Sprintf("EventID=%s", m[1]))
		case incEventIDRangeRegex.MatchString(c):
			m := incEventIDRangeRegex.FindStringSubmatch(c)
			r1, _ := strconv.Atoi(m[1])
			r2, _ := strconv.Atoi(m[2])
			if r1 >= r2 {
				return fmt.Errorf("event ID range '%s' is invalid", c)
			}
			includes = append(includes,
				fmt.Sprintf("(EventID &gt;= %d and EventID &lt;= %d)", r1, r2))
		case excEventIDRangeRegex.MatchString(c):
			m := excEventIDRangeRegex.FindStringSubmatch(c)
			r1, _ := strconv.Atoi(m[1])
			r2, _ := strconv.Atoi(m[2])
			if r1 >= r2 {
				return fmt.Errorf("event ID range '%s' is invalid", c)
			}
			excludes = append(excludes,
				fmt.Sprintf("(EventID &gt;= %d and EventID &lt;= %d)", r1, r2))
		default:
			return fmt.Errorf("invalid event ID query component ('%s')", c)
		}
	}

	actualLim := queryClauseLimit - len(q.Provider)
	if q.IgnoreOlder > 0 {
		actualLim--
	}
	if q.Level != "" {
		actualLim--
	}
	// we split selects in chunks of at most queryClauseLim size
	for i := 0; i < len(includes); i += actualLim {
		end := i + actualLim
		if end > len(includes) {
			end = len(includes)
		}
		chunk := includes[i:end]

		if len(chunk) == 1 {
			qp.selectEventFilters = append(qp.selectEventFilters, chunk...)
		} else if len(chunk) > 1 {
			qp.selectEventFilters = append(qp.selectEventFilters, "("+strings.Join(chunk, " or ")+")")
		}
	}

	if len(excludes) > 0 {
		qp.Suppress = "(" + strings.Join(excludes, " or ") + ")"
	}

	return nil
}

// withLevel returns a xpath selector for the event Level. The returned
// selector will select events with levels less than or equal to the specified
// level. Note that level 0 is used as a catch-all/unknown level.
//
// Accepted levels:
//
//	verbose           - 5
//	information, info - 4 or 0
//	warning,     warn - 3
//	error,       err  - 2
//	critical,    crit - 1
func (qp *queryParams) withLevel(q Query) error {
	if q.Level == "" {
		return nil
	}

	l := func(level int) string { return fmt.Sprintf("Level = %d", level) }

	var levelSelect []string
	for _, expr := range strings.Split(q.Level, ",") {
		expr = strings.TrimSpace(expr)
		switch strings.ToLower(expr) {
		default:
			return fmt.Errorf("invalid level ('%s') for query", q.Level)
		case "verbose", "5":
			levelSelect = append(levelSelect, l(5))
		case "information", "info", "4":
			levelSelect = append(levelSelect, l(0), l(4))
		case "warning", "warn", "3":
			levelSelect = append(levelSelect, l(3))
		case "error", "err", "2":
			levelSelect = append(levelSelect, l(2))
		case "critical", "crit", "1":
			levelSelect = append(levelSelect, l(1))
		case "0":
			levelSelect = append(levelSelect, l(0))
		}
	}

	if len(levelSelect) > 0 {
		qp.level = "(" + strings.Join(levelSelect, " or ") + ")"
	}

	return nil
}

func (qp *queryParams) withProvider(q Query) {
	if len(q.Provider) == 0 {
		return
	}

	selects := make([]string, 0, len(q.Provider))
	for _, p := range q.Provider {
		selects = append(selects, fmt.Sprintf("@Name='%s'", p))
	}

	qp.provider = fmt.Sprintf("Provider[%s]", strings.Join(selects, " or "))
}

func (qp *queryParams) buildSelects() {
	if len(qp.selectEventFilters) == 0 {
		sel := appendIfNotEmpty(qp.ignoreOlder, qp.level, qp.provider)
		if len(sel) == 0 {
			return
		}
		qp.Select = append(qp.Select, sel)
		return
	}
	for _, f := range qp.selectEventFilters {
		qp.Select = append(qp.Select, appendIfNotEmpty(qp.ignoreOlder, f, qp.level, qp.provider))
	}
}

func appendIfNotEmpty(ss ...string) []string {
	var sel []string
	for _, s := range ss {
		if s != "" {
			sel = append(sel, s)
		}
	}
	return sel
}

// executeTemplate populates a template with the given data and returns the
// value as a string.
func executeTemplate(t *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
