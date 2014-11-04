package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestThriftIdl_thriftReadFiles(t *testing.T) {

	if testing.Verbose() {
		LogInit(LOG_DEBUG, "", false, []string{"thrift", "thriftdetailed"})
	}

	f, _ := ioutil.TempFile("", "")
	defer os.Remove(f.Name())

	f.WriteString(`
/* simple test */
service Test {
	i32 add(1:i32 num1, 2: i32 num2)
}
	`)
	f.Close()

	thrift_files, err := ReadFiles([]string{f.Name()})
	if err != nil {
		t.Error("ReadFiles:", err)
	}
	if len(thrift_files) == 0 {
		t.Errorf("Did not read any files")
	}

	methods_map := BuildMethodsMap(thrift_files)
	if len(methods_map) == 0 {
		t.Error("Empty methods_map")
	}
	m, exists := methods_map["add"]
	if !exists || m.Service == nil || m.Method == nil ||
		m.Service.Name != "Test" || m.Method.Name != "add" {

		t.Error("Bad data:", m)
	}
}
