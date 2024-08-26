package application_pool

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"time"

	"math/rand"
)

func RandomInt() int {
	return rand.New(rand.NewSource(time.Now().UnixNano())).Int()
}

func Run(commands string) (*string, *string, error) {
	rInt := RandomInt()
	filename := fmt.Sprintf("command-%d.ps1", rInt)
	err := os.WriteFile(filename, []byte(commands), os.FileMode(0700))
	if err != nil {
		return nil, nil, fmt.Errorf("error writing command file: %+q", err)
	}

	var stderr bytes.Buffer
	var stdout bytes.Buffer

	// TODO: we could remove the need for a file by running these commands via WinRM, maybe?
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-NoLogo", "-NonInteractive", "-NoProfile", "-File", filename)

	cmd.Stderr = &stderr
	cmd.Stdout = &stdout

	defer os.Remove(filename)

	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("error starting: %+v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, nil, fmt.Errorf("error waiting: %+v", err)
	}

	stdOutStr := stdout.String()
	stdErrStr := stderr.String()

	return &stdOutStr, &stdErrStr, nil
}
