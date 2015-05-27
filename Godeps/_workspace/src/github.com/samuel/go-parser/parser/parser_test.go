package parser

import (
	"io"
	"reflect"
	"testing"
)

func testSimple(t *testing.T, name string, spec Spec, p Parser, in string, eok bool, exp interface{}) {
	st := &State{
		Input: NewStringInput(in),
		Spec:  spec,
	}
	out, ok, err := p(st)
	if err != nil {
		t.Fatalf("%s returned error %s (in '%s' exp '%+v')", name, err.Error(), in, exp)
	}
	if ok != eok {
		t.Fatalf("%s returned ok of %+v instead of %+v", name, ok, eok)
	}
	if !reflect.DeepEqual(out, exp) {
		t.Fatalf("%s returned '%+v' instead of '%+v'", name, out, exp)
	}
}

func TestAll(t *testing.T) {
	p := All(
		String("1"),
		String("test"),
	)
	testSimple(t, "All", Spec{}, p, "1test", true, "test")
	testSimple(t, "All", Spec{}, p, "1test222", true, "test")
}

func TestAny(t *testing.T) {
	p := Any(
		String("1"),
		String("test"),
	)
	testSimple(t, "Any", Spec{}, p, "2", false, nil)
	testSimple(t, "Any", Spec{}, p, "1", true, "1")
	testSimple(t, "Any", Spec{}, p, "test1", true, "test")
}

func TestMany(t *testing.T) {
	p := Many(
		String("1"),
	)
	testSimple(t, "Many", Spec{}, p, "", true, []interface{}{})
	testSimple(t, "Many", Spec{}, p, "11", true, []interface{}{"1", "1"})
	testSimple(t, "Many", Spec{}, p, "1122", true, []interface{}{"1", "1"})
}

func TestMany1(t *testing.T) {
	p := Many1(
		String("1"),
	)
	testSimple(t, "Many1", Spec{}, p, "3", false, nil)
	testSimple(t, "Many1", Spec{}, p, "11", true, []interface{}{"1", "1"})
	testSimple(t, "Many1", Spec{}, p, "1122", true, []interface{}{"1", "1"})
}

func TestString(t *testing.T) {
	p := String("test")
	testSimple(t, "String", Spec{}, p, "test", true, "test")
	testSimple(t, "String", Spec{}, p, "testaa", true, "test")
}

func TestComments(t *testing.T) {
	spec := Spec{
		CommentStart: "/*",
		CommentEnd:   "*/",
		CommentLine:  String("//"),
	}
	in := NewStringInput(`// this is a test
	    // only a test
	    /* this is
	       a multiline comment */`)
	st := &State{Spec: spec, Input: in}
	p := OneLineComment()
	out, d, err := p(st)
	if err != nil {
		t.Fatalf("OneLinecomment returned error %s", err.Error())
	}
	if !d {
		t.Fatal("OneLineComment returned !ok")
	}
	exp := " this is a test"
	if outString, ok := out.(string); !ok {
		t.Fatal("OneLinecomment returned non-string")
	} else if outString != exp {
		t.Fatalf("OneLineComment returned '%s' instead of '%s'", outString, exp)
	}
}

func TestStringInput(t *testing.T) {
	testString := "tes†ing mitä"

	in := NewStringInput(testString)
	outString := ""
	for {
		r, err := in.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("StringInput.Next returned error %s", err.Error())
		}
		in.Pop(1)
		outString += string(r)
	}

	if testString != outString {
		t.Fatalf("Next/Pop produced unmatched output of %#v instead of %#v", outString, testString)
	}

	in = NewStringInput(testString)
	in.Pop(1)
	outString, err := in.Get(5)
	if err != nil {
		t.Fatal("Get(5) returned error %s", err.Error())
	}
	if "es†in" != outString {
		t.Fatalf("Get produced unmatched output of %#v instead of %#v", outString, "es†in")
	}

	in = NewStringInput(testString)
	outString, err = in.Get(12)
	if err != nil {
		t.Fatal("Get(12) returned error %s", err.Error())
	}
	if testString != outString {
		t.Fatalf("Get(len) produced unmatched output of %#v instead of %#v", outString, testString)
	}
	outString, err = in.Get(13)
	if err != io.EOF {
		t.Fatal("Get(len+1) returned error %+v but should have returned EOF", err)
	}
}
