package ceph

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func Execute(c *CephConfig, command string) (string, error) {

	cmdArgs := []string{}
	cmdArgsDefault := []string{"--conf", c.ConfigPath, "--name", c.User, "--format", "json"}
	cmdArgs = append(cmdArgs, cmdArgsDefault...)
	cmdArgs = append(cmdArgs, strings.Split(command, " ")...)

	var output string

	if strings.Contains(c.BinaryPath, "docker") {
		cmdCeph := []string{"/usr/bin/ceph"}
		cmdDockerExec := append(cmdCeph, cmdArgs...)
		output = ExecInContainer(cmdDockerExec)
	} else {

		cmd := exec.Command(c.BinaryPath, cmdArgs...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("[Error running ceph status command] %v: %s", command, stderr.String())
		}

		output = out.String()
	}

	// Ceph doesn't sanitize its output, and may return invalid JSON.  Patch this
	// up for them, as having some inaccurate data is better than none.
	output = strings.Replace(output, "-inf", "0", -1)
	output = strings.Replace(output, "inf", "0", -1)

	return output, nil
}
