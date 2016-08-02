package exec

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/actions/lookup/lutool"
)

type execRunner struct {
	formatter   *fmtstr.EventFormatString
	config      execRunnerConfig
	credentials *syscall.Credential
}

func init() {
	processors.RegisterPlugin("lookup.exec", newExecLookupProcessor)
}

func newExecLookupProcessor(cfg common.Config) (processors.Processor, error) {
	config := defaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	runner, err := newExecRunner(config.Runner)
	if err != nil {
		return nil, err
	}

	keys := config.Key
	if len(keys) == 0 {
		keys = runner.Keys()
	}

	keyer, err := lutool.MakeKeyBuilder(keys)
	if err != nil {
		return nil, err
	}

	return lutool.NewCachedLookupTool(
		"lookup.exec",
		config.Cache,
		keyer,
		runner,
	)
}

func newExecRunner(config execRunnerConfig) (*execRunner, error) {
	fs := config.Run
	if fs.NumFields() == 0 {
		logp.Err("Exec runner without event arguments")
		return nil, errors.New("no arguments")
	}

	credential, err := loadCredentials(config.User)
	if err != nil {
		return nil, err
	}

	runner := &execRunner{
		formatter:   fs,
		credentials: credential,
		config:      config,
	}
	return runner, nil
}

func (r *execRunner) Keys() []string {
	return r.formatter.Fields()
}

func (r *execRunner) Exec(event common.MapStr) (common.MapStr, error) {
	// XXX: if lookup.keys has been configured and keys can extract the field values,
	//       but extracting arguments from event fails (e.g. due to missing fields in
	//       arguments not used by key), No meta-data will be cached + execRunner will
	//       not be re-executed.

	command, err := r.formatter.Run(event)
	if err != nil {
		return nil, err
	}

	args, err := parseCommand(command)
	if err != nil {
		logp.Err("Failed to parse command '%v'", command)
		return nil, err
	}

	// TODO: add support for event being pushed on stdin instead of passing arguments
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = r.config.WorkingDir
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Chroot:     r.config.Chroot,
		Credential: r.credentials,
	}
	cmd.Env = r.config.Env

	// read and capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var timer *time.Timer
	if r.config.Timeout > 0 {
		timer = time.AfterFunc(r.config.Timeout, func() {
			// TODO: kill enough or send SIGKILL on unix based systems?
			// XXX: this might not kill spawned child processes running in background.
			cmd.Process.Kill()
		})
	}

	var fields map[string]interface{}
	var decodeErr error
	var wg sync.WaitGroup
	stderrBuf := bytes.NewBuffer(nil)

	wg.Add(1)
	go func() {
		defer wg.Done()
		decodeErr = json.NewDecoder(stdout).Decode(&fields)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		limit := int64(2048)
		n, err := io.CopyN(stderrBuf, stderr, limit)
		if limit == n && err != nil {
			io.Copy(ioutil.Discard, stderr)
		}
	}()

	err = cmd.Wait()
	if timer != nil {
		timer.Stop()
	}
	wg.Wait()
	if err != nil {
		// something bad happened, let's try to generate some error message
		if stderrBuf.Len() == 0 {
			return nil, err
		}

		return nil, fmt.Errorf("%v with stderr '%v'", err.Error(), stderrBuf.String())
	}

	// check decoder failed:
	if decodeErr != nil {
		return nil, decodeErr
	}

	return common.MapStr(fields), nil
}

func parseCommand(in string) ([]string, error) {
	trim := func(s string) string { return strings.TrimFunc(s, unicode.IsSpace) }

	raw := trim(in)
	if len(raw) == 0 {
		return nil, fmt.Errorf("Empty command given")
	}

	var args []string
	for len(raw) > 0 {
		idx := strings.IndexFunc(raw, func(r rune) bool {
			return r == '"' || unicode.IsSpace(r)
		})
		if idx < 0 {
			args = append(args, trim(raw))
			break
		}

		switch raw[idx] {
		case '"':
			if str := trim(raw[:idx]); str != "" {
				args = append(args, str)
			}
			raw = raw[idx+1:]
			idx = strings.IndexRune(raw, '"')
			if idx < 0 {
				return nil, fmt.Errorf("Failed parsing '%v' due to missing '\"'", in)
			}
			args = append(args, raw[:idx])
			raw = trim(raw[idx+1:])
		default:
			args = append(args, trim(raw[:idx]))
			raw = trim(raw[idx+1:])
		}
	}
	return args, nil
}
