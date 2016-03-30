// Copyright (c) 2012 VMware, Inc.

package main

import (
	"fmt"
	"strings"

	"github.com/elastic/gosigar"
)

func main() {
	pids := gosigar.ProcList{}
	pids.Get()

	// ps -eo pid,ppid,stime,time,rss,user,state,command
	fmt.Print("  PID  PPID STIME     TIME    RSS USER            S COMMAND\n")

	for _, pid := range pids.List {
		state := gosigar.ProcState{}
		mem := gosigar.ProcMem{}
		time := gosigar.ProcTime{}
		args := gosigar.ProcArgs{}

		if err := state.Get(pid); err != nil {
			continue
		}
		if err := mem.Get(pid); err != nil {
			continue
		}
		if err := time.Get(pid); err != nil {
			continue
		}
		if err := args.Get(pid); err != nil {
			continue
		}

		fmt.Printf("%5d %5d %s %s %6d %-15s %c %s\n",
			pid, state.Ppid,
			time.FormatStartTime(), time.FormatTotal(),
			mem.Resident/1024, state.Username, state.State,
			strings.Join(args.List, " "))
	}
}
