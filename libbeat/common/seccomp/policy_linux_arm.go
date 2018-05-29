package seccomp

import "github.com/elastic/go-seccomp-bpf"

func init() {
	defaultPolicy = &seccomp.Policy{
		DefaultAction: seccomp.ActionAllow,
		Syscalls: []seccomp.SyscallGroup{
			{
				Action: seccomp.ActionErrno,
				Names: []string{
					"execve",
					"execveat",
					"fork",
					"vfork",
				},
			},
		},
	}
}
