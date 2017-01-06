package ceph

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func Execute(c *CephConfig, command string) (string, error) {
	cmdArgs := []string{"--conf", c.ConfigPath, "--name", c.User, "--format", "json"}
	cmdArgs = append(cmdArgs, strings.Split(command, " ")...)

	cmd := exec.Command(c.BinaryPath, cmdArgs...)

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("[Error running ceph status command] %v: %s", command, err)
	}

	output := out.String()

	// Ceph doesn't sanitize its output, and may return invalid JSON.  Patch this
	// up for them, as having some inaccurate data is better than none.
	output = strings.Replace(output, "-inf", "0", -1)
	output = strings.Replace(output, "inf", "0", -1)

	return output, nil
}
