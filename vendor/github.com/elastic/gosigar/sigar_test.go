// +build linux darwin windows

package sigar_test

import (
	"os"
	"os/user"
	"testing"

	sigar "github.com/elastic/gosigar"
)

func TestProcStateUsername(t *testing.T) {
	proc := sigar.ProcState{}
	err := proc.Get(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	user, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}

	if user.Username != proc.Username {
		t.Fatalf("Usernames don't match, expected %s, but got %s",
			user.Username, proc.Username)
	}
}
