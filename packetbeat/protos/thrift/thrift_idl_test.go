package thrift

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/elastic/beats/libbeat/logp"
)

func thriftIdlForTesting(t *testing.T, content string) *ThriftIdl {
	f, _ := ioutil.TempFile("", "")
	defer os.Remove(f.Name())

	f.WriteString(content)
	f.Close()

	idl, err := NewThriftIdl([]string{f.Name()})
	if err != nil {
		t.Fatal("Parsing failed:", err)
	}

	return idl
}

func TestThriftIdl_thriftReadFiles(t *testing.T) {

	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"thrift", "thriftdetailed"})
	}

	idl := thriftIdlForTesting(t, `
/* simple test */
service Test {
       i32 add(1:i32 num1, 2: i32 num2)
}
`)

	methods_map := idl.MethodsByName
	if len(methods_map) == 0 {
		t.Error("Empty methods_map")
	}
	m, exists := methods_map["add"]
	if !exists || m.Service == nil || m.Method == nil ||
		m.Service.Name != "Test" || m.Method.Name != "add" {

		t.Error("Bad data:", m)
	}
	if *m.Params[1] != "num1" || *m.Params[2] != "num2" {
		t.Error("Bad params", m.Params)
	}
	if len(m.Exceptions) != 0 {
		t.Error("Non empty exceptions", m.Exceptions)
	}
}
