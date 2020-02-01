package hjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// EncoderOptions defines options for encoding to Hjson.
type EncoderOptions struct {
	// End of line, should be either \n or \r\n
	Eol string
	// Place braces on the same line
	BracesSameLine bool
	// Deprecated: Hjson always emits braces
	EmitRootBraces bool
	// Always place string in quotes
	QuoteAlways bool
	// Indent string
	IndentBy string
	// Allow the -0 value (unlike ES6)
	AllowMinusZero bool
	// Encode unknown values as 'null'
	UnknownAsNull bool
}

// DefaultOptions returns the default encoding options.
func DefaultOptions() EncoderOptions {
	opt := EncoderOptions{}
	opt.Eol = "\n"
	opt.BracesSameLine = false
	opt.EmitRootBraces = true
	opt.QuoteAlways = false
	opt.IndentBy = "  "
	opt.AllowMinusZero = false
	opt.UnknownAsNull = false
	return opt
}

type hjsonEncoder struct {
	bytes.Buffer // output
	EncoderOptions
	indent int
}

var needsEscape, needsQuotes, needsEscapeML, startsWithKeyword, needsEscapeName *regexp.Regexp

func init() {
	var commonRange = `\x7f-\x9f\x{00ad}\x{0600}-\x{0604}\x{070f}\x{17b4}\x{17b5}\x{200c}-\x{200f}\x{2028}-\x{202f}\x{2060}-\x{206f}\x{feff}\x{fff0}-\x{ffff}`
	// needsEscape tests if the string can be written without escapes
	needsEscape = regexp.MustCompile(`[\\\"\x00-\x1f` + commonRange + `]`)
	// needsQuotes tests if the string can be written as a quoteless string (includes needsEscape but without \\ and \")
	needsQuotes = regexp.MustCompile(`^\s|^"|^'|^#|^/\*|^//|^\{|^\}|^\[|^\]|^:|^,|\s$|[\x00-\x1f\x7f-\x9f\x{00ad}\x{0600}-\x{0604}\x{070f}\x{17b4}\x{17b5}\x{200c}-\x{200f}\x{2028}-\x{202f}\x{2060}-\x{206f}\x{feff}\x{fff0}-\x{ffff}]`)
	// needsEscapeML tests if the string can be written as a multiline string (like needsEscape but without \n, \r, \\, \", \t)
	var x08Or9 = `\x08` // `\x09` for the old behavior
	needsEscapeML = regexp.MustCompile(`'''|^[\s]+$|[\x00-` + x08Or9 + `\x0b\x0c\x0e-\x1f` + commonRange + `]`)
	// starts with a keyword and optionally is followed by a comment
	startsWithKeyword = regexp.MustCompile(`^(true|false|null)\s*((,|\]|\}|#|//|/\*).*)?$`)
	needsEscapeName = regexp.MustCompile(`[,\{\[\}\]\s:#"']|//|/\*`)
}

var meta = map[byte][]byte{
	// table of character substitutions
	'\b': []byte("\\b"),
	'\t': []byte("\\t"),
	'\n': []byte("\\n"),
	'\f': []byte("\\f"),
	'\r': []byte("\\r"),
	'"':  []byte("\\\""),
	'\\': []byte("\\\\"),
}

func (e *hjsonEncoder) quoteReplace(text string) string {
	return string(needsEscape.ReplaceAllFunc([]byte(text), func(a []byte) []byte {
		c := meta[a[0]]
		if c != nil {
			return c
		}
		r, _ := utf8.DecodeRune(a)
		return []byte(fmt.Sprintf("\\u%04x", r))
	}))
}

func (e *hjsonEncoder) quote(value string, separator string, isRootObject bool) {

	// Check if we can insert this string without quotes
	// see hjson syntax (must not parse as true, false, null or number)

	if len(value) == 0 {
		e.WriteString(separator + `""`)
	} else if e.QuoteAlways ||
		needsQuotes.MatchString(value) ||
		startsWithNumber([]byte(value)) ||
		startsWithKeyword.MatchString(value) {

		// If the string contains no control characters, no quote characters, and no
		// backslash characters, then we can safely slap some quotes around it.
		// Otherwise we first check if the string can be expressed in multiline
		// format or we must replace the offending characters with safe escape
		// sequences.

		if !needsEscape.MatchString(value) {
			e.WriteString(separator + `"` + value + `"`)
		} else if !needsEscapeML.MatchString(value) && !isRootObject {
			e.mlString(value, separator)
		} else {
			e.WriteString(separator + `"` + e.quoteReplace(value) + `"`)
		}
	} else {
		// return without quotes
		e.WriteString(separator + value)
	}
}

func (e *hjsonEncoder) mlString(value string, separator string) {
	// wrap the string into the ''' (multiline) format

	a := strings.Split(strings.Replace(value, "\r", "", -1), "\n")

	if len(a) == 1 {
		// The string contains only a single line. We still use the multiline
		// format as it avoids escaping the \ character (e.g. when used in a
		// regex).
		e.WriteString(separator + "'''")
		e.WriteString(a[0])
	} else {
		e.writeIndent(e.indent + 1)
		e.WriteString("'''")
		for _, v := range a {
			indent := e.indent + 1
			if len(v) == 0 {
				indent = 0
			}
			e.writeIndent(indent)
			e.WriteString(v)
		}
		e.writeIndent(e.indent + 1)
	}
	e.WriteString("'''")
}

func (e *hjsonEncoder) quoteName(name string) string {
	if len(name) == 0 {
		return `""`
	}

	// Check if we can insert this name without quotes

	if needsEscapeName.MatchString(name) {
		if needsEscape.MatchString(name) {
			name = e.quoteReplace(name)
		}
		return `"` + name + `"`
	}
	// without quotes
	return name
}

type sortAlpha []reflect.Value

func (s sortAlpha) Len() int {
	return len(s)
}
func (s sortAlpha) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortAlpha) Less(i, j int) bool {
	return s[i].String() < s[j].String()
}

func (e *hjsonEncoder) writeIndent(indent int) {
	e.WriteString(e.Eol)
	for i := 0; i < indent; i++ {
		e.WriteString(e.IndentBy)
	}
}

func (e *hjsonEncoder) useMarshaler(value reflect.Value, separator string) error {
	b, err := value.Interface().(json.Marshaler).MarshalJSON()
	if err != nil {
		return err
	}
	e.WriteString(separator)
	e.WriteString(string(b))
	return nil
}

var marshaler = reflect.TypeOf((*json.Marshaler)(nil)).Elem()

func (e *hjsonEncoder) str(value reflect.Value, noIndent bool, separator string, isRootObject bool) error {

	// Produce a string from value.

	kind := value.Kind()

	if kind == reflect.Interface || kind == reflect.Ptr {
		if value.IsNil() {
			e.WriteString(separator)
			e.WriteString("null")
			return nil
		}
		value = value.Elem()
		kind = value.Kind()
	}

	if value.Type().Implements(marshaler) {
		return e.useMarshaler(value, separator)
	}

	switch kind {
	case reflect.String:
		e.quote(value.String(), separator, isRootObject)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.WriteString(separator)
		e.WriteString(strconv.FormatInt(value.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr:
		e.WriteString(separator)
		e.WriteString(strconv.FormatUint(value.Uint(), 10))

	case reflect.Float32, reflect.Float64:
		// JSON numbers must be finite. Encode non-finite numbers as null.
		e.WriteString(separator)
		number := value.Float()
		if math.IsInf(number, 0) || math.IsNaN(number) {
			e.WriteString("null")
		} else if !e.AllowMinusZero && number == -0 {
			e.WriteString("0")
		} else {
			// find shortest representation ('G' does not work)
			val := strconv.FormatFloat(number, 'f', -1, 64)
			exp := strconv.FormatFloat(number, 'E', -1, 64)
			if len(exp) < len(val) {
				val = strings.ToLower(exp)
			}
			e.WriteString(val)
		}

	case reflect.Bool:
		e.WriteString(separator)
		if value.Bool() {
			e.WriteString("true")
		} else {
			e.WriteString("false")
		}

	case reflect.Slice, reflect.Array:

		len := value.Len()
		if len == 0 {
			e.WriteString(separator)
			e.WriteString("[]")
			break
		}

		indent1 := e.indent
		e.indent++

		if !noIndent && !e.BracesSameLine {
			e.writeIndent(indent1)
		} else {
			e.WriteString(separator)
		}
		e.WriteString("[")

		// Join all of the element texts together, separated with newlines
		for i := 0; i < len; i++ {
			e.writeIndent(e.indent)
			if err := e.str(value.Index(i), true, "", false); err != nil {
				return err
			}
		}

		e.writeIndent(indent1)
		e.WriteString("]")

		e.indent = indent1

	case reflect.Map:

		len := value.Len()
		if len == 0 {
			e.WriteString(separator)
			e.WriteString("{}")
			break
		}

		indent1 := e.indent
		e.indent++
		if !noIndent && !e.BracesSameLine {
			e.writeIndent(indent1)
		} else {
			e.WriteString(separator)
		}
		e.WriteString("{")

		keys := value.MapKeys()
		sort.Sort(sortAlpha(keys))

		// Join all of the member texts together, separated with newlines
		for i := 0; i < len; i++ {
			e.writeIndent(e.indent)
			e.WriteString(e.quoteName(keys[i].String()))
			e.WriteString(":")
			if err := e.str(value.MapIndex(keys[i]), false, " ", false); err != nil {
				return err
			}
		}

		e.writeIndent(indent1)
		e.WriteString("}")
		e.indent = indent1

	case reflect.Struct:

		l := value.NumField()
		if l == 0 {
			e.WriteString(separator)
			e.WriteString("{}")
			break
		}

		indent1 := e.indent
		e.indent++
		if !noIndent && !e.BracesSameLine {
			e.writeIndent(indent1)
		} else {
			e.WriteString(separator)
		}
		e.WriteString("{")

		// Join all of the member texts together, separated with newlines
		for i := 0; i < l; i++ {
			curStructField := value.Type().Field(i)
			curField := value.Field(i)

			name := curStructField.Name
			jsonTag := curStructField.Tag.Get("json")
			jsonComment := curStructField.Tag.Get("comment")
			omitEmpty := false
			if jsonTag == "-" {
				continue
			}
			splits := strings.Split(jsonTag, ",")
			if splits[0] != "" {
				name = splits[0]
			}
			if len(splits) > 1 {
				for _, opt := range splits[1:] {
					if opt == "omitempty" {
						omitEmpty = true
					}
				}
			}
			if omitEmpty && isEmptyValue(curField) {
				continue
			}
			if len(jsonComment) > 0 {
				for _, line := range strings.Split(jsonComment, e.Eol) {
					e.WriteString(separator)
					e.writeIndent(e.indent)
					e.WriteString(fmt.Sprintf("# %s", line))
					e.WriteString(separator)
				}
			}
			e.writeIndent(e.indent)
			e.WriteString(e.quoteName(name))
			e.WriteString(":")
			if err := e.str(curField, false, " ", false); err != nil {
				return err
			}
			if len(jsonComment) > 0 && i < l-1 {
				e.WriteString(e.Eol)
			}
		}

		e.writeIndent(indent1)
		e.WriteString("}")

		e.indent = indent1

	default:
		if e.UnknownAsNull {
			// Use null as a placeholder for non-JSON values.
			e.WriteString("null")
		} else {
			return errors.New("Unsupported type " + value.Type().String())
		}
	}
	return nil
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	default:
		return false
	}
}

// Marshal returns the Hjson encoding of v using
// default options.
//
// See MarshalWithOptions.
//
func Marshal(v interface{}) ([]byte, error) {
	return MarshalWithOptions(v, DefaultOptions())
}

// MarshalWithOptions returns the Hjson encoding of v.
//
// Marshal traverses the value v recursively.
//
// Boolean values encode as JSON booleans.
//
// Floating point, integer, and Number values encode as JSON numbers.
//
// String values encode as Hjson strings (quoteless, multiline or
// JSON).
//
// Array and slice values encode as JSON arrays.
//
// Map values encode as JSON objects. The map's key type must be a
// string. The map keys are sorted and used as JSON object keys.
//
// Pointer values encode as the value pointed to.
// A nil pointer encodes as the null JSON value.
//
// Interface values encode as the value contained in the interface.
// A nil interface value encodes as the null JSON value.
//
// JSON cannot represent cyclic data structures and Marshal does not
// handle them. Passing cyclic structures to Marshal will result in
// an infinite recursion.
//
func MarshalWithOptions(v interface{}, options EncoderOptions) ([]byte, error) {
	e := &hjsonEncoder{}
	e.indent = 0
	e.Eol = options.Eol
	e.BracesSameLine = options.BracesSameLine
	e.QuoteAlways = options.QuoteAlways
	e.IndentBy = options.IndentBy

	err := e.str(reflect.ValueOf(v), true, "", true)
	if err != nil {
		return nil, err
	}
	return e.Bytes(), nil
}
