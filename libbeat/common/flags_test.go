package common

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestStringArrFlag(t *testing.T) {
	tests := []struct {
		init     []string
		def      string
		in       []string
		expected []string
	}{
		{nil, "test", nil, []string{"test"}},
		{nil, "test", []string{"new"}, []string{"new"}},
		{nil, "test", []string{"a", "b"}, []string{"a", "b"}},
		{[]string{"default"}, "newdefault", nil, []string{"newdefault"}},
		{[]string{"default"}, "newdefault", []string{"arg"}, []string{"arg"}},
		{[]string{"default"}, "newdefault", []string{"a", "b"}, []string{"a", "b"}},
		{[]string{"default"}, "newdefault", []string{"a", "b", "a", "b"}, []string{"a", "b"}},
	}

	for _, test := range tests {
		test := test
		name := fmt.Sprintf("init=%v,default=%v,in=%v,out=%v", test.init, test.def, test.in, test.expected)

		t.Run(name, func(t *testing.T) {
			init := make([]string, len(test.init))
			copy(init, test.init)

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			flag := StringArrVarFlag(fs, &init, "a", "add")

			if test.def != "" {
				flag.SetDefault(test.def)
			}

			defaultValue := flag.String()

			goflagUsage, _ := withStderr(fs.PrintDefaults)
			goflagExpectedUsage := fmt.Sprintf("  -a value\n    \tadd (default %v)\n", defaultValue)

			cmd := cobra.Command{}
			cmd.PersistentFlags().AddGoFlag(fs.Lookup("a"))
			cobraUsage := cmd.LocalFlags().FlagUsages()
			cobraExpectedUsage := fmt.Sprintf("  -a, --a string   add (default \"%v\")\n", defaultValue)

			for _, v := range test.in {
				err := flag.Set(v)
				if err != nil {
					t.Error(err)
				}
			}

			assert.Equal(t, goflagExpectedUsage, goflagUsage)
			assert.Equal(t, cobraExpectedUsage, cobraUsage)
			assert.Equal(t, test.expected, init)
			assert.Equal(t, test.expected, flag.List())
		})
	}
}

func TestSettingsFlag(t *testing.T) {
	tests := []struct {
		in       []string
		expected map[string]interface{}
	}{
		{nil, nil},
		{[]string{"a=1"}, map[string]interface{}{"a": uint64(1)}},
		{[]string{"a=1", "b=false"}, map[string]interface{}{"a": uint64(1), "b": false}},
		{[]string{"a=1", "b"}, map[string]interface{}{"a": uint64(1), "b": true}},
		{[]string{"a=1", "c=${a}"}, map[string]interface{}{"a": uint64(1), "c": uint64(1)}},
	}

	for _, test := range tests {
		test := test
		name := strings.Join(test.in, ",")

		t.Run(name, func(t *testing.T) {
			config := NewConfig()
			f := NewSettingsFlag(config)

			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.Var(f, "s", "message")

			goflagUsage, _ := withStderr(fs.PrintDefaults)
			goflagExpectedUsage := "  -s value\n    \tmessage\n"

			cmd := cobra.Command{}
			cmd.PersistentFlags().AddGoFlag(fs.Lookup("s"))
			cobraUsage := cmd.LocalFlags().FlagUsages()
			cobraExpectedUsage := "  -s, --s setting=value   message\n"

			for _, in := range test.in {
				err := f.Set(in)
				if err != nil {
					t.Error(err)
				}
			}

			var result map[string]interface{}
			err := config.Unpack(&result)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, goflagExpectedUsage, goflagUsage)
			assert.Equal(t, cobraExpectedUsage, cobraUsage)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestOverwriteFlag(t *testing.T) {
	config, err := NewConfigFrom(map[string]interface{}{
		"a": "test",
	})
	if err != nil {
		panic(err)
	}

	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	ConfigOverwriteFlag(fs, config, "a", "a", "", "message")

	goflagUsage, _ := withStderr(fs.PrintDefaults)
	goflagExpectedUsage := "  -a value\n    \tmessage\n"
	assert.Equal(t, goflagExpectedUsage, goflagUsage)

	cmd := cobra.Command{}
	cmd.PersistentFlags().AddGoFlag(fs.Lookup("a"))
	cobraUsage := cmd.LocalFlags().FlagUsages()
	cobraExpectedUsage := "  -a, --a string   message\n"
	assert.Equal(t, cobraExpectedUsage, cobraUsage)

	fs.Set("a", "overwrite")
	final, err := config.String("a", -1)
	assert.NoError(t, err)
	assert.Equal(t, "overwrite", final)
}

// capture stderr and return captured string
func withStderr(fn func()) (string, error) {
	stderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stderr = w
	defer func() {
		os.Stderr = stderr
	}()

	outC := make(chan string)
	go func() {
		// capture all output
		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		r.Close()
		outC <- buf.String()
	}()

	fn()
	w.Close()
	result := <-outC
	return result, err
}
