package reviewdog

import (
	"context"
	"os/exec"
	"sync"
)

var _ DiffService = &DiffString{}

type DiffString struct {
	b     []byte
	strip int
}

func NewDiffString(diff string, strip int) DiffService {
	return &DiffString{b: []byte(diff), strip: strip}
}

func (d *DiffString) Diff(_ context.Context) ([]byte, error) {
	return d.b, nil
}

func (d *DiffString) Strip() int {
	return d.strip
}

var _ DiffService = &DiffCmd{}

type DiffCmd struct {
	cmd   *exec.Cmd
	strip int
	out   []byte
	done  bool
	mu    sync.RWMutex
}

func NewDiffCmd(cmd *exec.Cmd, strip int) *DiffCmd {
	return &DiffCmd{cmd: cmd, strip: strip}
}

// Diff returns diff. It caches the result and can be used more than once.
func (d *DiffCmd) Diff(_ context.Context) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.done {
		return d.out, nil
	}
	out, err := d.cmd.Output()
	// Exit status of `git diff` is 1 if diff exists, so ignore the error if diff
	// presents.
	if err != nil && len(out) == 0 {
		return nil, err
	}
	d.out = out
	d.done = true
	return d.out, nil
}

func (d *DiffCmd) Strip() int {
	return d.strip
}
