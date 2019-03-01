package readfile

import "fmt"

type LineTerminator uint8

const (
	InvalidTerminator LineTerminator = iota
	LineFeed
	VerticalTab
	FormFeed
	CarriageReturn
	CarriageReturnLineFeed
	NextLine
	LineSeparator
	ParagraphSeparator
)

var (
	lineTerminators = map[string]LineTerminator{
		"line_feed":                 LineFeed,
		"vertical_tab":              VerticalTab,
		"form_feed":                 FormFeed,
		"carriage_return":           CarriageReturn,
		"carriage_return_line_feed": CarriageReturnLineFeed,
		"next_line":                 NextLine,
		"line_separator":            LineSeparator,
		"paragraph_separator":       ParagraphSeparator,
	}

	lineTerminatorCharacters = map[LineTerminator][]byte{
		LineFeed:               []byte{'\u000A'},
		VerticalTab:            []byte{'\u000B'},
		FormFeed:               []byte{'\u000C'},
		CarriageReturn:         []byte{'\u000D'},
		CarriageReturnLineFeed: []byte("\u000C\u000A"),
		NextLine:               []byte{'\u0085'},
		LineSeparator:          []byte("\u2028"),
		ParagraphSeparator:     []byte("\u2029"),
	}
)

func (l *LineTerminator) Unpack(option string) error {
	terminator, ok := lineTerminators[option]
	if !ok {
		return fmt.Errorf("invalid line terminator: %s", option)
	}

	*l = terminator

	return nil
}
