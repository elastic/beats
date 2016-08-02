package exec

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/stretchr/testify/assert"
)

func TestExecInitFail(t *testing.T) {
	tests := []struct {
		title  string
		config execRunnerConfig
	}{
		{
			"no keys in format string",
			execRunnerConfig{
				Run: fmtstr.MustCompileEvent("test"),
			},
		},
		{
			"invalid username",
			execRunnerConfig{
				Run:  fmtstr.MustCompileEvent("test %{[key]}"),
				User: "non_existent_system_user_test",
			},
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		_, err := newExecRunner(test.config)
		assert.Error(t, err)
	}
}

func TestExec(t *testing.T) {
	mustHaveExec(t)

	tests := []struct {
		title    string
		command  string
		keys     []string
		event    common.MapStr
		expected common.MapStr
	}{
		{
			"run plain",
			`fields f1 %{[a]} %{[f2]} b`,
			[]string{"a", "f2"},
			common.MapStr{"a": "test", "f2": "ft"},
			common.MapStr{"f1": "test", "ft": "b"},
		},
		{
			"run with quoted string",
			`fields f1 "pre %{[a]} post" %{[f2]} "another field"`,
			[]string{"a", "f2"},
			common.MapStr{"a": "test", "f2": "ft"},
			common.MapStr{
				"f1": "pre test post",
				"ft": "another field",
			},
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		config := defaultConfig.Runner
		config.Run = fmtstr.MustCompileEvent(
			fmt.Sprintf("%s -test.run TestHelper -- %s", os.Args[0], test.command))
		config.Env = []string{
			"WANT_HELPER_PROCESS=1",
		}
		runner, err := newExecRunner(config)
		if err != nil {
			t.Error(err)
			continue
		}

		actual, err := runner.Exec(test.event)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.keys, runner.Keys())
		assert.Equal(t, test.expected, actual)
	}
}

func TestExecErr(t *testing.T) {
	tests := []struct {
		title   string
		command string
		workdir string
		event   common.MapStr
	}{
		{
			"test invalid command",
			`"invalid nonexistent shell tool" "%{[arg]}"`,
			"",
			common.MapStr{"arg": "arg"},
		},
		{
			"configure invalid work directory",
			os.Args[0] + " -test.run TestHelper -- echo %{[arg]}",
			"invalid nonexistent work directory",
			common.MapStr{"arg": "arg"},
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		config := defaultConfig.Runner
		config.Run = fmtstr.MustCompileEvent(test.command)
		config.WorkingDir = test.workdir
		config.Env = []string{
			"WANT_HELPER_PROCESS=1",
		}

		runner, err := newExecRunner(config)
		if err != nil {
			t.Error(err)
			continue
		}

		_, err = runner.Exec(test.event)
		assert.Error(t, err)
		t.Logf("process exited with: %v", err)
	}
}

func TestExecScriptErr(t *testing.T) {
	mustHaveExec(t)

	tests := []struct {
		title   string
		command string
		timeout time.Duration
		event   common.MapStr
		error   string
	}{
		{
			"do not run if argument key is missing",
			"%{[cmd]}",
			0,
			common.MapStr{"key": "wrong"},
			"",
		},
		{
			"process exits with error",
			"%{[cmd]} %{[code]}",
			0,
			common.MapStr{"cmd": "exit", "code": 10},
			"exit status 10",
		},
		{
			"process exists without output",
			"exit %{[code]}",
			0,
			common.MapStr{"code": 0},
			"EOF",
		},
		{
			"fail due to invalid json being printed",
			"echo %{[key]}",
			0,
			common.MapStr{"key": "value"},
			"",
		},
		{
			"fail with message to stderr",
			"echoerr %{[key]}",
			0,
			common.MapStr{"key": "errormessage"},
			"exit status 1 with stderr 'errormessage'",
		},
		{
			"timeout fail",
			"sleep %{[duration]}",
			100 * time.Millisecond,
			common.MapStr{"duration": "10s"},
			"signal: killed",
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		config := defaultConfig.Runner
		config.Run = fmtstr.MustCompileEvent(
			fmt.Sprintf("%s -test.run TestHelper -- %s", os.Args[0], test.command))
		config.Env = []string{
			"WANT_HELPER_PROCESS=1",
		}
		if test.timeout > 0 {
			config.Timeout = test.timeout
		}

		runner, err := newExecRunner(config)
		if err != nil {
			t.Error(err)
			continue
		}

		_, err = runner.Exec(test.event)
		if test.error != "" {
			assert.EqualError(t, err, test.error)
		} else {
			assert.Error(t, err)
		}

		t.Logf("process exited with: %v", err)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		title    string
		command  string
		expected []string
	}{
		{
			"empty command fails",
			"",
			nil,
		},
		{
			"non-closed escape fails",
			`abc "def`,
			nil,
		},
		{
			"parse with arguments",
			"command arg1 arg2 arg3",
			[]string{"command", "arg1", "arg2", "arg3"},
		},
		{
			"parse escaped all args",
			`command "a 1" "a 2" "a 3"`,
			[]string{"command", "a 1", "a 2", "a 3"},
		},
		{
			"parse escaped first arg",
			`command "a 1" a2 a3`,
			[]string{"command", "a 1", "a2", "a3"},
		},
		{
			"parse escaped last arg",
			`command a1 a2 "a 3"`,
			[]string{"command", "a1", "a2", "a 3"},
		},
		{
			"parse escaped middle arg",
			`command a1 "a 2" a3`,
			[]string{"command", "a1", "a 2", "a3"},
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		cmd, err := parseCommand(test.command)
		if test.expected == nil {
			assert.Error(t, err)
			continue
		}
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.expected, cmd)
	}
}

// TestHelper isn't a real test. It's used as a helper process
func TestHelper(*testing.T) {
	if os.Getenv("WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "fields":
		kv := [2][]string{}
		i := 0
		for _, arg := range args {
			kv[i] = append(kv[i], arg)
			i = 1 - i
		}

		L := len(kv[0])
		if len(kv[1]) < L {
			L = len(kv[1])
		}

		event := common.MapStr{}
		for i := range kv[0] {
			event[kv[0][i]] = kv[1][i]
		}

		dat, err := json.Marshal(event)
		if err != nil {
			fmt.Fprint(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(dat))

	case "echo":
		iargs := []interface{}{}
		for _, s := range args {
			iargs = append(iargs, s)
		}
		fmt.Println(iargs...)

	case "echoerr":
		iargs := []interface{}{}
		for _, s := range args {
			iargs = append(iargs, s)
		}
		fmt.Fprint(os.Stderr, iargs...)
		os.Exit(1)

	case "exit":
		n, _ := strconv.Atoi(args[0])
		os.Exit(n)

	case "sleep":
		d, err := time.ParseDuration(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		time.Sleep(d)

	case "cat":
		if len(args) == 0 {
			io.Copy(os.Stdout, os.Stdin)
			return
		}
		exit := 0
		for _, fn := range args {
			f, err := os.Open(fn)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				exit = 2
			} else {
				defer f.Close()
				io.Copy(os.Stdout, f)
			}
		}
		os.Exit(exit)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
		os.Exit(2)
	}
}

func mustHaveExec(t *testing.T) {
	if !hasExec() {
		t.Skipf("skipping test: cannot exec subprocess on %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

// HasExec reports whether the current system can start new processes
// using os.StartProcess or (more commonly) exec.Command.
func hasExec() bool {
	switch runtime.GOOS {
	case "nacl":
		return false
	case "darwin":
		if strings.HasPrefix(runtime.GOARCH, "arm") {
			return false
		}
	}
	return true
}
