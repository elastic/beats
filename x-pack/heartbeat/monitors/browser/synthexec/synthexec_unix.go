package synthexec

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

func init() {
	platformCmdMutate = func(cmd *exec.Cmd) {
		// Note that while cmd.SysProcAtr takes a syscall.SysProcAttr object
		// we are passing in a unix.SysProcAttr object
		// this is equivalent, but the unix package is not considered deprecated
		// as the syscall package is
		cmd.SysProcAttr = &unix.SysProcAttr{
			// Ensure node subprocesses are killed if this process dies (linux only)
			Pdeathsig: unix.SIGKILL,
		}
	}
}
