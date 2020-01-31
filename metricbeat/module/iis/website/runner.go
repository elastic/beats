package website

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type Website struct {
	Name            string `json:"name"`
	ApplicationPool string `json:"applicationPool"`
	PhysicalPath    string `json:"physicalPath"`
	State           string `json:"state"`
}

func Run(commands string) (*string, error) {

	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd := exec.Command("powershell.exe", "-ExecutionPolicy", "Bypass", "-NoLogo", "-NonInteractive", "-NoProfile", "-Command", commands)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("error starting: %+v", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("error waiting: %+v", err)
	}
	stdOutStr := stdout.String()

	return &stdOutStr, nil
}

func GetWorkerProcessId(appPool string) (int, error) {

	commands := fmt.Sprintf(`
Import-Module WebAdministration
$pids = dir "IIS:\AppPools\%s\WorkerProcesses" | Select-Object -expand processId | ConvertTo-Json

if ($pids.Count -eq 0) {
    Write-Host "[]"
} else {
    if ($pids.Count -gt 1) {
      $v = "[""{0}""]" -f $pids[0].ToString()
        Write-Host $v
    } else { 
 $v = $pids | ConvertTo-Json
	    Write-Host $v
    }
}
  `, appPool)

	stdout, err := Run(commands)
	if err != nil {
		return 0, fmt.Errorf("error retrieving Worker Process ID's for App Pool %q: %+v", appPool, err)
	}
	var stringId string
	err = json.Unmarshal([]byte(*stdout), &stringId)
	if err != nil {
		return 0, fmt.Errorf("error parsing %q as a worker process id of type int: %+v", stdout, err)
	}
	id, err := strconv.Atoi(stringId)
	return id, nil
}

func GetApplicationPool(host string) (string, error) {
	commands := fmt.Sprintf(`
Import-Module WebAdministration
Get-Website -Name %q | ConvertTo-Json -Compress
  `, host)

	stdout, err := Run(commands)
	if err != nil {
		return "", fmt.Errorf("error retrieving website: %+v", err)
	}

	var site Website
	if out := stdout; out != nil && *out != "" {
		v := *out
		err := json.Unmarshal([]byte(v), &site)
		if err != nil {
			return "", fmt.Errorf("error unmarshalling website %q: %+v", host, err)
		}
	}

	if site.Name == "" {
		return "", fmt.Errorf("website %q was not found", host)
	}
	if site.State == "Stopped" {
		return "", fmt.Errorf("website %q has stopped", host)
	}

	return site.ApplicationPool, nil
}
