// +build ignore

// This is a simple wrapper to set the GOARCH environment variable, execute the
// given command, and output the stdout to a file. It's the unix equivalent of
// GOARCH=arch command > output.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	goarch = flag.String("goarch", "", "GOARCH value")
	output = flag.String("output", "", "output file")
	cmd    = flag.String("cmd", "", "command to run")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	// Run command.
	parts := strings.Fields(*cmd)
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Env = os.Environ()

	if *goarch != "" {
		cmd.Env = append(cmd.Env, "GOARCH="+*goarch)
	}

	if *output != "" {
		outputBytes, err := cmd.Output()
		if err != nil {
			return err
		}

		// Write output.
		f, err := os.Create(*output)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(outputBytes)
		return err
	}

	cmd.Stdout = cmd.Stdout
	return cmd.Run()
}
