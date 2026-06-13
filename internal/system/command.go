package system

import (
	"os/exec"
)

// RunCommand runs a command and returns its combined stdout/stderr output as a string.
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
