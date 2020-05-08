package reviewdog

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"

	"github.com/reviewdog/errorformat"
	"github.com/reviewdog/errorformat/fmts"
)

// ParserOpt represents option to create Parser. Either FormatName or
// Errorformat should be specified.
type ParserOpt struct {
	FormatName  string
	Errorformat []string
}

// NewParser returns Parser based on ParserOpt.
func NewParser(opt *ParserOpt) (Parser, error) {
	name := opt.FormatName

	if name != "" && len(opt.Errorformat) > 0 {
		return nil, errors.New("you cannot specify both format name and errorformat at the same time")
	}

	if name == "checkstyle" {
		return NewCheckStyleParser(), nil
	}
	// use defined errorformat
	if name != "" {
		efm, ok := fmts.DefinedFmts()[name]
		if !ok {
			return nil, fmt.Errorf("%q is not supported. consider to add new errrorformat to https://github.com/reviewdog/errorformat", name)
		}
		opt.Errorformat = efm.Errorformat
	}
	if len(opt.Errorformat) == 0 {
		return nil, errors.New("errorformat is empty")
	}
	return NewErrorformatParserString(opt.Errorformat)
}

var _ Parser = &ErrorformatParser{}

// ErrorformatParser is errorformat parser.
type ErrorformatParser struct {
	efm *errorformat.Errorformat
}

// NewErrorformatParser returns a new ErrorformatParser.
func NewErrorformatParser(efm *errorformat.Errorformat) *ErrorformatParser {
	return &ErrorformatParser{efm: efm}
}

// NewErrorformatParserString returns a new ErrorformatParser from errorformat
// in string representation.
func NewErrorformatParserString(efms []string) (*ErrorformatParser, error) {
	efm, err := errorformat.NewErrorformat(efms)
	if err != nil {
		return nil, err
	}
	return NewErrorformatParser(efm), nil
}

func (p *ErrorformatParser) Parse(r io.Reader) ([]*CheckResult, error) {
	s := p.efm.NewScanner(r)
	var rs []*CheckResult
	for s.Scan() {
		e := s.Entry()
		if e.Valid {
			rs = append(rs, &CheckResult{
				Path:    e.Filename,
				Lnum:    e.Lnum,
				Col:     e.Col,
				Message: e.Text,
				Lines:   e.Lines,
			})
		}
	}
	return rs, nil
}

var _ Parser = &CheckStyleParser{}

// CheckStyleParser is checkstyle parser.
type CheckStyleParser struct{}

// NewCheckStyleParser returns a new CheckStyleParser.
func NewCheckStyleParser() Parser {
	return &CheckStyleParser{}
}

func (p *CheckStyleParser) Parse(r io.Reader) ([]*CheckResult, error) {
	var cs = new(CheckStyleResult)
	if err := xml.NewDecoder(r).Decode(cs); err != nil {
		return nil, err
	}
	var rs []*CheckResult
	for _, file := range cs.Files {
		for _, cerr := range file.Errors {
			rs = append(rs, &CheckResult{
				Path:    file.Name,
				Lnum:    cerr.Line,
				Col:     cerr.Column,
				Message: cerr.Message,
				Lines: []string{
					fmt.Sprintf("%v:%d:%d: %v: %v (%v)",
						file.Name, cerr.Line, cerr.Column, cerr.Severity, cerr.Message, cerr.Source),
				},
			})
		}
	}
	return rs, nil
}

// CheckStyleResult represents checkstyle XML result.
// <?xml version="1.0" encoding="utf-8"?><checkstyle version="4.3"><file ...></file>...</checkstyle>
//
// References:
//   - http://checkstyle.sourceforge.net/
//   - http://eslint.org/docs/user-guide/formatters/#checkstyle
type CheckStyleResult struct {
	XMLName xml.Name          `xml:"checkstyle"`
	Version string            `xml:"version,attr"`
	Files   []*CheckStyleFile `xml:"file,omitempty"`
}

// CheckStyleFile represents <file name="fname"><error ... />...</file>
type CheckStyleFile struct {
	Name   string             `xml:"name,attr"`
	Errors []*CheckStyleError `xml:"error"`
}

// CheckStyleError represents <error line="1" column="10" severity="error" message="msg" source="src" />
type CheckStyleError struct {
	Column   int    `xml:"column,attr,omitempty"`
	Line     int    `xml:"line,attr"`
	Message  string `xml:"message,attr"`
	Severity string `xml:"severity,attr,omitempty"`
	Source   string `xml:"source,attr,omitempty"`
}
