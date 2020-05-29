// Package errorformat provides 'errorformat' functionality of Vim. :h
// errorformat
package errorformat

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// Errorformat provides errorformat feature.
type Errorformat struct {
	Efms []*Efm
}

// Scanner provides a interface for scanning compiler/linter/static analyzer
// result using Errorformat.
type Scanner struct {
	*Errorformat
	source *bufio.Scanner

	qi *qfinfo

	entry   *Entry // entry which is returned by Entry() func
	mlpoped bool   // is multiline entry poped (for non-end multiline entry)
}

// NewErrorformat compiles given errorformats string (efms) and returns a new
// Errorformat. It returns error if the errorformat is invalid.
func NewErrorformat(efms []string) (*Errorformat, error) {
	errorformat := &Errorformat{Efms: make([]*Efm, 0, len(efms))}
	for _, efm := range efms {
		e, err := NewEfm(efm)
		if err != nil {
			return nil, err
		}
		errorformat.Efms = append(errorformat.Efms, e)
	}
	return errorformat, nil
}

// NewScanner returns a new Scanner to read from r.
func (errorformat *Errorformat) NewScanner(r io.Reader) *Scanner {
	return &Scanner{
		Errorformat: errorformat,
		source:      bufio.NewScanner(r),
		qi:          &qfinfo{},
		mlpoped:     true,
	}
}

type qfinfo struct {
	filestack   []string
	currfile    string
	dirstack    []string
	directory   string
	multiscan   bool
	multiline   bool
	multiignore bool

	qflist []*Entry
}

type qffields struct {
	namebuf   string
	errmsg    string
	lnum      int
	col       int
	useviscol bool
	pattern   string
	enr       int
	etype     byte
	valid     bool

	lines []string
}

// Entry represents matched entry of errorformat, equivalent to Vim's quickfix
// list item.
type Entry struct {
	// name of a file
	Filename string `json:"filename"`
	// line number
	Lnum int `json:"lnum"`
	// column number (first column is 1)
	Col int `json:"col"`
	// true: "col" is visual column
	// false: "col" is byte index
	Vcol bool `json:"vcol"`
	// error number
	Nr int `json:"nr"`
	// search pattern used to locate the error
	Pattern string `json:"pattern"`
	// description of the error
	Text string `json:"text"`
	// type of the error, 'E', '1', etc.
	Type rune `json:"type"`
	// true: recognized error message
	Valid bool `json:"valid"`

	// Original error lines (often one line. more than one line for multi-line
	// errorformat. :h errorformat-multi-line)
	Lines []string `json:"lines"`
}

// || message
// /path/to/file|| message
// /path/to/file|1| message
// /path/to/file|1 col 14| message
// /path/to/file|1 col 14 error 8| message
// {filename}|{lnum}[ col {col}][ {type} [{nr}]]| {text}
func (e *Entry) String() string {
	s := fmt.Sprintf("%s|", e.Filename)
	if e.Lnum > 0 {
		s += strconv.Itoa(e.Lnum)
	}
	if e.Col > 0 {
		s += fmt.Sprintf(" col %d", e.Col)
	}
	if t := e.Types(); t != "" {
		s += " " + t
	}
	s += "|"
	if e.Text != "" {
		s += " " + e.Text
	}
	return s
}

// Types makes a nice message out of the error character and the error number:
//
// qf_types in src/quickfix.c
func (e *Entry) Types() string {
	s := ""
	switch e.Type {
	case 'e', 'E':
		s = "error"
	case 0:
		if e.Nr > 0 {
			s = "error"
		}
	case 'w', 'W':
		s = "warning"
	case 'i', 'I':
		s = "info"
	default:
		s = string(e.Type)
	}
	if e.Nr > 0 {
		if s != "" {
			s += " "
		}
		s += strconv.Itoa(e.Nr)
	}
	return s
}

// Scan advances the Scanner to the next entry matched with errorformat, which
// will then be available through the Entry method. It returns false
// when the scan stops by reaching the end of the input.
func (s *Scanner) Scan() bool {
	for s.source.Scan() {
		line := s.source.Text()
		status, fields := s.parseLine(line)
		switch status {
		case qffail:
			continue
		case qfendmultiline:
			s.mlpoped = true
			s.entry = s.qi.qflist[len(s.qi.qflist)-1]
			return true
		case qfignoreline:
			continue
		}
		var lastml *Entry // last multiline entry which isn't poped out
		if !s.mlpoped {
			lastml = s.qi.qflist[len(s.qi.qflist)-1]
		}
		qfl := &Entry{
			Filename: fields.namebuf,
			Lnum:     fields.lnum,
			Col:      fields.col,
			Nr:       fields.enr,
			Pattern:  fields.pattern,
			Text:     fields.errmsg,
			Vcol:     fields.useviscol,
			Valid:    fields.valid,
			Type:     rune(fields.etype),
			Lines:    fields.lines,
		}
		if qfl.Filename == "" && s.qi.currfile != "" {
			qfl.Filename = s.qi.currfile
		}
		s.qi.qflist = append(s.qi.qflist, qfl)
		if s.qi.multiline {
			s.mlpoped = false // mark multiline entry is not poped
			// if there is last multiline entry which isn't poped out yet, pop it out now.
			if lastml != nil {
				s.entry = lastml
				return true
			}
			continue
		}
		// multiline flag doesn't be reset with new entry.
		// %Z or nomach are the only way to reset multiline flag.
		s.entry = qfl
		return true
	}
	// pop last not-ended multiline entry
	if !s.mlpoped {
		s.mlpoped = true
		s.entry = s.qi.qflist[len(s.qi.qflist)-1]
		return true
	}
	return false
}

// Entry returns the most recent entry generated by a call to Scan.
func (s *Scanner) Entry() *Entry {
	return s.entry
}

type qfstatus int

const (
	qffail qfstatus = iota
	qfignoreline
	qfendmultiline
	qfok
)

func (s *Scanner) parseLine(line string) (qfstatus, *qffields) {
	return s.parseLineInternal(line, 0)
}

func (s *Scanner) parseLineInternal(line string, i int) (qfstatus, *qffields) {
	fields := &qffields{valid: true, enr: -1, lines: []string{line}}
	tail := ""
	var idx byte
	nomatch := false
	var efm *Efm
	for ; i <= len(s.Efms); i++ {
		if i == len(s.Efms) {
			nomatch = true
			break
		}
		efm = s.Efms[i]

		idx = efm.prefix
		if s.qi.multiscan && strchar("OPQ", idx) {
			continue
		}

		if (idx == 'C' || idx == 'Z') && !s.qi.multiline {
			continue
		}

		r := efm.Match(line)
		if r == nil {
			continue
		}

		if strchar("EWI", idx) {
			fields.etype = idx
		}

		if r.F != "" { // %f
			fields.namebuf = r.F
			if strchar("OPQ", idx) && !fileexists(fields.namebuf) {
				continue
			}
		}
		fields.enr = r.N  // %n
		fields.lnum = r.L // %l
		fields.col = r.C  // %c
		if r.T != 0 {
			fields.etype = r.T // %t
		}
		if efm.flagplus && !s.qi.multiscan { // %+
			fields.errmsg = line
		} else if r.M != "" {
			fields.errmsg = r.M
		}
		tail = r.R     // %r
		if r.P != "" { // %p
			fields.useviscol = true
			fields.col = 0
			for _, m := range r.P {
				fields.col++
				if m == '\t' {
					fields.col += 7
					fields.col -= fields.col % 8
				}
			}
			fields.col++ // last pointer (e.g. ^)
		}
		if r.V != 0 {
			fields.useviscol = true
			fields.col = r.V
		}
		if r.S != "" {
			fields.pattern = fmt.Sprintf("^%v$", regexp.QuoteMeta(r.S))
		}
		break
	}
	s.qi.multiscan = false
	if nomatch || idx == 'D' || idx == 'X' {
		if !nomatch {
			if idx == 'D' {
				if fields.namebuf == "" {
					return qffail, nil
				}
				s.qi.directory = fields.namebuf
				s.qi.dirstack = append(s.qi.dirstack, s.qi.directory)
			} else if idx == 'X' && len(s.qi.dirstack) > 0 {
				s.qi.directory = s.qi.dirstack[len(s.qi.dirstack)-1]
				s.qi.dirstack = s.qi.dirstack[:len(s.qi.dirstack)-1]
			}
		}
		fields.namebuf = ""
		fields.lnum = 0
		fields.valid = false
		fields.errmsg = line
		if nomatch {
			s.qi.multiline = false
			s.qi.multiignore = false
		}
	} else if !nomatch {
		if strchar("AEWI", idx) {
			s.qi.multiline = true    // start of a multi-line message
			s.qi.multiignore = false // reset continuation
		} else if strchar("CZ", idx) {
			// continuation of multi-line msg
			if !s.qi.multiignore {
				qfprev := s.qi.qflist[len(s.qi.qflist)-1]
				qfprev.Lines = append(qfprev.Lines, line)
				if qfprev == nil {
					return qffail, nil
				}
				if fields.errmsg != "" && !s.qi.multiignore {
					if qfprev.Text == "" {
						qfprev.Text = fields.errmsg
					} else {
						qfprev.Text += "\n" + fields.errmsg
					}
				}
				if qfprev.Nr < 1 {
					qfprev.Nr = fields.enr
				}
				if fields.etype != 0 && qfprev.Type == 0 {
					qfprev.Type = rune(fields.etype)
				}
				if qfprev.Lnum == 0 {
					qfprev.Lnum = fields.lnum
				}
				if qfprev.Col == 0 {
					qfprev.Col = fields.col
				}
				qfprev.Vcol = fields.useviscol
			}
			if idx == 'Z' {
				s.qi.multiline = false
				s.qi.multiignore = false
				return qfendmultiline, fields
			}
			return qfignoreline, nil
		} else if strchar("OPQ", idx) {
			// global file names
			fields.valid = false
			if fields.namebuf == "" || fileexists(fields.namebuf) {
				if fields.namebuf != "" && idx == 'P' {
					s.qi.currfile = fields.namebuf
					s.qi.filestack = append(s.qi.filestack, s.qi.currfile)
				} else if idx == 'Q' && len(s.qi.filestack) > 0 {
					s.qi.currfile = s.qi.filestack[len(s.qi.filestack)-1]
					s.qi.filestack = s.qi.filestack[:len(s.qi.filestack)-1]
				}
				fields.namebuf = ""
				if tail != "" {
					s.qi.multiscan = true
					return s.parseLineInternal(strings.TrimLeft(tail, " \t"), i)
				}
			}
		}
		if efm.flagminus { // generally exclude this line
			if s.qi.multiline { // also exclude continuation lines
				s.qi.multiignore = true
			}
			return qfignoreline, nil
		}
	}
	return qfok, fields
}

// Efm represents a errorformat.
type Efm struct {
	regex *regexp.Regexp

	flagplus  bool
	flagminus bool
	prefix    byte
}

var fmtpattern = map[byte]string{
	'f': `(?P<f>(?:[[:alpha:]]:)?(?:\\ |[^ ])+?)`,
	'n': `(?P<n>\d+)`,
	'l': `(?P<l>\d+)`,
	'c': `(?P<c>\d+)`,
	't': `(?P<t>.)`,
	'm': `(?P<m>.+)`,
	'r': `(?P<r>.*)`,
	'p': `(?P<p>[- 	.]*)`,
	'v': `(?P<v>\d+)`,
	's': `(?P<s>.+)`,
}

// NewEfm converts a 'errorformat' string to regular expression pattern with
// flags and returns Efm.
//
// quickfix.c: efm_to_regpat
func NewEfm(errorformat string) (*Efm, error) {
	var regpat bytes.Buffer
	var efmp byte
	var i = 0
	var incefmp = func() {
		i++
		efmp = errorformat[i]
	}
	efm := &Efm{}
	regpat.WriteRune('^')
	for ; i < len(errorformat); i++ {
		efmp = errorformat[i]
		if efmp == '%' {
			incefmp()
			// - do not support %>
			if re, ok := fmtpattern[efmp]; ok {
				regpat.WriteString(re)
			} else if efmp == '*' {
				incefmp()
				if efmp == '[' || efmp == '\\' {
					regpat.WriteByte(efmp)
					if efmp == '[' { // %*[^a-z0-9] etc.
						incefmp()
						for efmp != ']' {
							regpat.WriteByte(efmp)
							if i == len(errorformat)-1 {
								return nil, errors.New("E374: Missing ] in format string")
							}
							incefmp()
						}
						regpat.WriteByte(efmp)
					} else { // %*\D, %*\s etc.
						incefmp()
						regpat.WriteByte(efmp)
					}
					regpat.WriteRune('+')
				} else {
					return nil, fmt.Errorf("E375: Unsupported %%%v in format string", string(efmp))
				}
			} else if (efmp == '+' || efmp == '-') &&
				i < len(errorformat)-1 &&
				strchar("DXAEWICZGOPQ", errorformat[i+1]) {
				if efmp == '+' {
					efm.flagplus = true
					incefmp()
				} else if efmp == '-' {
					efm.flagminus = true
					incefmp()
				}
				efm.prefix = efmp
			} else if strchar(`%\.^$?+[`, efmp) {
				// regexp magic characters
				regpat.WriteByte(efmp)
			} else if efmp == '#' {
				regpat.WriteRune('*')
			} else {
				if strchar("DXAEWICZGOPQ", efmp) {
					efm.prefix = efmp
				} else {
					return nil, fmt.Errorf("E376: Invalid %%%v in format string prefix", string(efmp))
				}
			}
		} else { // copy normal character
			if efmp == '\\' && i < len(errorformat)-1 {
				incefmp()
			} else if strchar(`.+*()|[{^$`, efmp) { // escape regexp atoms
				regpat.WriteRune('\\')
			}
			regpat.WriteByte(efmp)
		}
	}
	regpat.WriteRune('$')
	re, err := regexp.Compile(regpat.String())
	if err != nil {
		return nil, err
	}
	efm.regex = re
	return efm, nil
}

// Match represents match of Efm. ref: Basic items in :h errorformat
type Match struct {
	F string // (%f) file name
	N int    // (%n) error number
	L int    // (%l) line number
	C int    // (%c) column number
	T byte   // (%t) error type
	M string // (%m) error message
	R string // (%r) the "rest" of a single-line file message
	P string // (%p) pointer line
	V int    // (%v) virtual column number
	S string // (%s) search text
}

// Match returns match against given string.
func (efm *Efm) Match(s string) *Match {
	ms := efm.regex.FindStringSubmatch(s)
	if len(ms) == 0 {
		return nil
	}
	match := &Match{}
	names := efm.regex.SubexpNames()
	for i, name := range names {
		if i == 0 {
			continue
		}
		m := ms[i]
		switch name {
		case "f":
			match.F = m
		case "n":
			match.N = mustAtoI(m)
		case "l":
			match.L = mustAtoI(m)
		case "c":
			match.C = mustAtoI(m)
		case "t":
			match.T = m[0]
		case "m":
			match.M = m
		case "r":
			match.R = m
		case "p":
			match.P = m
		case "v":
			match.V = mustAtoI(m)
		case "s":
			match.S = m
		}
	}
	return match
}

func strchar(chars string, c byte) bool {
	return bytes.ContainsAny([]byte{c}, chars)
}

func mustAtoI(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

// Vim sees the file exists or not (maybe for quickfix usage), but do not see
// file exists this implementation. Always return true.
var fileexists = func(filename string) bool {
	return true
	// _, err := os.Stat(filename)
	// return err == nil
}
