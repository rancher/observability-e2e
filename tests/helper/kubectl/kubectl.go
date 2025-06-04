package kubectl

import (
	"bytes"
	"errors"
	"os/exec"
)

// Execute runs a kubectl command with the given arguments,
// automatically appending --insecure-skip-tls-verify=true unless it's already present.
func Execute(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", errors.New(stderr.String())
	}

	return stdout.String(), nil
}
