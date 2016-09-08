package gocsv

import (
	"bytes"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
	"testing"
)

func Test_readTo(t *testing.T) {
	b := bytes.NewBufferString(`foo,BAR,Baz
f,1,baz
e,3,b`)
	d := &decoder{in: b}

	var samples []Sample
	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 sample instances, got %d", len(samples))
	}
	expected := Sample{Foo: "f", Bar: 1, Baz: "baz"}
	if expected != samples[0] {
		t.Fatalf("expected first sample %v, got %v", expected, samples[0])
	}
	expected = Sample{Foo: "e", Bar: 3, Baz: "b"}
	if expected != samples[1] {
		t.Fatalf("expected second sample %v, got %v", expected, samples[1])
	}

	b = bytes.NewBufferString(`foo,BAR,Baz
f,1,baz
e,BAD_INPUT,b`)
	d = &decoder{in: b}
	samples = []Sample{}
	err := readTo(d, &samples)
	if err == nil {
		t.Fatalf("Expected error from bad input, got: %+v", samples)
	}
	switch actualErr := err.(type) {
	case *csv.ParseError:
		if actualErr.Line != 3 {
			t.Fatalf("Expected csv.ParseError on line 3, got: %d", actualErr.Line)
		}
		if actualErr.Column != 2 {
			t.Fatalf("Expected csv.ParseError in column 2, got: %d", actualErr.Column)
		}
	default:
		t.Fatalf("incorrect error type: %T", err)
	}

}

func Test_readTo_complex_embed(t *testing.T) {
	b := bytes.NewBufferString(`first,foo,BAR,Baz,last,abc
aa,bb,11,cc,dd,ee
ff,gg,22,hh,ii,jj`)
	d := &decoder{in: b}

	var samples []SkipFieldSample
	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 sample instances, got %d", len(samples))
	}
	expected := SkipFieldSample{
		EmbedSample: EmbedSample{
			Qux: "aa",
			Sample: Sample{
				Foo: "bb",
				Bar: 11,
				Baz: "cc",
			},
			Quux: "dd",
		},
		Corge: "ee",
	}
	if expected != samples[0] {
		t.Fatalf("expected first sample %v, got %v", expected, samples[0])
	}
	expected = SkipFieldSample{
		EmbedSample: EmbedSample{
			Qux: "ff",
			Sample: Sample{
				Foo: "gg",
				Bar: 22,
				Baz: "hh",
			},
			Quux: "ii",
		},
		Corge: "jj",
	}
	if expected != samples[1] {
		t.Fatalf("expected first sample %v, got %v", expected, samples[1])
	}
}

func Test_readEach(t *testing.T) {
	b := bytes.NewBufferString(`first,foo,BAR,Baz,last,abc
aa,bb,11,cc,dd,ee
ff,gg,22,hh,ii,jj`)
	d := &decoder{in: b}

	c := make(chan SkipFieldSample)
	var samples []SkipFieldSample
	go func() {
		if err := readEach(d, c); err != nil {
			t.Fatal(err)
		}
	}()
	for v := range c {
		samples = append(samples, v)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 sample instances, got %d", len(samples))
	}
	expected := SkipFieldSample{
		EmbedSample: EmbedSample{
			Qux: "aa",
			Sample: Sample{
				Foo: "bb",
				Bar: 11,
				Baz: "cc",
			},
			Quux: "dd",
		},
		Corge: "ee",
	}
	if expected != samples[0] {
		t.Fatalf("expected first sample %v, got %v", expected, samples[0])
	}
	expected = SkipFieldSample{
		EmbedSample: EmbedSample{
			Qux: "ff",
			Sample: Sample{
				Foo: "gg",
				Bar: 22,
				Baz: "hh",
			},
			Quux: "ii",
		},
		Corge: "jj",
	}
	if expected != samples[1] {
		t.Fatalf("expected first sample %v, got %v", expected, samples[1])
	}
}

func Test_maybeMissingStructFields(t *testing.T) {
	structTags := []fieldInfo{
		{keys: []string{"foo"}},
		{keys: []string{"bar"}},
		{keys: []string{"baz"}},
	}
	badHeaders := []string{"hi", "mom", "bacon"}
	goodHeaders := []string{"foo", "bar", "baz"}

	// no tags to match, expect no error
	if err := maybeMissingStructFields([]fieldInfo{}, goodHeaders); err != nil {
		t.Fatal(err)
	}

	// bad headers, expect an error
	if err := maybeMissingStructFields(structTags, badHeaders); err == nil {
		t.Fatal("expected an error, but no error found")
	}

	// good headers, expect no error
	if err := maybeMissingStructFields(structTags, goodHeaders); err != nil {
		t.Fatal(err)
	}

	// extra headers, but all structtags match; expect no error
	moarHeaders := append(goodHeaders, "qux", "quux", "corge", "grault")
	if err := maybeMissingStructFields(structTags, moarHeaders); err != nil {
		t.Fatal(err)
	}

	// not all structTags match, but there's plenty o' headers; expect
	// error
	mismatchedHeaders := []string{"foo", "qux", "quux", "corgi"}
	if err := maybeMissingStructFields(structTags, mismatchedHeaders); err == nil {
		t.Fatal("expected an error, but no error found")
	}
}

func Test_maybeDoubleHeaderNames(t *testing.T) {
	b := bytes.NewBufferString(`foo,BAR,foo
f,1,baz
e,3,b`)
	d := &decoder{in: b}
	var samples []Sample

	// *** check maybeDoubleHeaderNames
	if err := maybeDoubleHeaderNames([]string{"foo", "BAR", "foo"}); err == nil {
		t.Fatal("maybeDoubleHeaderNames did not raise an error when a should have.")
	}

	// *** check readTo
	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}
	// Double header allowed, value should be of third row
	if samples[0].Foo != "baz" {
		t.Fatal("Double header allowed, value should be of third row but is not. Function called is readTo.")
	}
	// Double header not allowed, should fail
	FailIfDoubleHeaderNames = true
	if err := readTo(d, &samples); err == nil {
		t.Fatal("Double header not allowed but no error raised. Function called is readTo.")
	}

	// *** check readEach
	FailIfDoubleHeaderNames = false
	b = bytes.NewBufferString(`foo,BAR,foo
	f,1,baz
	e,3,b`)
	d = &decoder{in: b}
	samples = samples[:0]
	c := make(chan Sample)
	go func() {
		if err := readEach(d, c); err != nil {
			t.Fatal(err)
		}
	}()
	for v := range c {
		samples = append(samples, v)
	}
	// Double header allowed, value should be of third row
	if samples[0].Foo != "baz" {
		t.Fatal("Double header allowed, value should be of third row but is not. Function called is readEach.")
	}
	// Double header not allowed, should fail
	FailIfDoubleHeaderNames = true
	b = bytes.NewBufferString(`foo,BAR,foo
f,1,baz
e,3,b`)
	d = &decoder{in: b}
	c = make(chan Sample)
	go func() {
		if err := readEach(d, c); err == nil {
			t.Fatal("Double header not allowed but no error raised. Function called is readEach.")
		}
	}()
	for v := range c {
		samples = append(samples, v)
	}
}

func TestUnmarshalToCallback(t *testing.T) {
	b := bytes.NewBufferString(`first,foo,BAR,Baz,last,abc
aa,bb,11,cc,dd,ee
ff,gg,22,hh,ii,jj`)
	var samples []SkipFieldSample
	if err := UnmarshalBytesToCallback(b.Bytes(), func(s SkipFieldSample) {
		samples = append(samples, s)
	}); err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 {
		t.Fatalf("expected 2 sample instances, got %d", len(samples))
	}
	expected := SkipFieldSample{
		EmbedSample: EmbedSample{
			Qux: "aa",
			Sample: Sample{
				Foo: "bb",
				Bar: 11,
				Baz: "cc",
			},
			Quux: "dd",
		},
		Corge: "ee",
	}
	if expected != samples[0] {
		t.Fatalf("expected first sample %v, got %v", expected, samples[0])
	}
	expected = SkipFieldSample{
		EmbedSample: EmbedSample{
			Qux: "ff",
			Sample: Sample{
				Foo: "gg",
				Bar: 22,
				Baz: "hh",
			},
			Quux: "ii",
		},
		Corge: "jj",
	}
	if expected != samples[1] {
		t.Fatalf("expected first sample %v, got %v", expected, samples[1])
	}
}

// TestRenamedTypes tests for unmarshaling functions on redefined basic types.
func TestRenamedTypesUnmarshal(t *testing.T) {
	b := bytes.NewBufferString(`foo;bar
1,4;1.5
2,3;2.4`)
	d := &decoder{in: b}
	var samples []RenamedSample

	// Set different csv field separator to enable comma in floats
	SetCSVReader(func(in io.Reader) *csv.Reader {
		csvin := csv.NewReader(in)
		csvin.Comma = ';'
		return csvin
	})
	// Switch back to default for tests executed after this
	defer SetCSVReader(DefaultCSVReader)

	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}
	if samples[0].RenamedFloatUnmarshaler != 1.4 {
		t.Fatalf("Parsed float value wrong for renamed float64 type. Expected 1.4, got %v.", samples[0].RenamedFloatUnmarshaler)
	}
	if samples[0].RenamedFloatDefault != 1.5 {
		t.Fatalf("Parsed float value wrong for renamed float64 type without an explicit unmarshaler function. Expected 1.5, got %v.", samples[0].RenamedFloatDefault)
	}

	// Test that errors raised by UnmarshalCSV are correctly reported
	b = bytes.NewBufferString(`foo;bar
4.2;2.4`)
	d = &decoder{in: b}
	samples = samples[:0]
	if perr, _ := readTo(d, &samples).(*csv.ParseError); perr == nil {
		t.Fatalf("Expected ParseError, got nil.")
	} else if _, ok := perr.Err.(UnmarshalError); !ok {
		t.Fatalf("Expected UnmarshalError, got %v", perr.Err)
	}
}

func (rf *RenamedFloat64Unmarshaler) UnmarshalCSV(csv string) (err error) {
	// Purely for testing purposes: Raise error on specific string
	if csv == "4.2" {
		return UnmarshalError{"Test error: Invalid float 4.2"}
	}

	// Convert , to . before parsing to create valid float strings
	converted := strings.Replace(csv, ",", ".", -1)
	var f float64
	if f, err = strconv.ParseFloat(converted, 64); err != nil {
		return err
	}
	*rf = RenamedFloat64Unmarshaler(f)
	return nil
}

type UnmarshalError struct {
	msg string
}

func (e UnmarshalError) Error() string {
	return e.msg
}

func TestMultipleStructTags(t *testing.T) {
	b := bytes.NewBufferString(`foo,BAR,Baz
e,3,b`)
	d := &decoder{in: b}

	var samples []MultiTagSample
	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}
	if samples[0].Foo != "b" {
		t.Fatalf("expected second tag value 'b' in multi tag struct field, got %v", samples[0].Foo)
	}

	b = bytes.NewBufferString(`foo,BAR
e,3`)
	d = &decoder{in: b}

	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}
	if samples[0].Foo != "e" {
		t.Fatalf("wrong value in multi tag struct field, expected 'e', got %v", samples[0].Foo)
	}

	b = bytes.NewBufferString(`BAR,Baz
3,b`)
	d = &decoder{in: b}

	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}
	if samples[0].Foo != "b" {
		t.Fatal("wrong value in multi tag struct field")
	}
}

func TestStructTagSeparator(t *testing.T) {
	b := bytes.NewBufferString(`foo,BAR,Baz
e,3,b`)
	d := &decoder{in: b}

	defaultTagSeparator := TagSeparator
	TagSeparator = "|"
	defer func() { TagSeparator = defaultTagSeparator }()

	var samples []TagSeparatorSample
	if err := readTo(d, &samples); err != nil {
		t.Fatal(err)
	}

	if samples[0].Foo != "b" {
		t.Fatal("expected second tag value in multi tag struct field.")
	}
}
