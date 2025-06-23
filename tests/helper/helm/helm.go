package helm

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// runCommandWithKubeconfig runs a command with the given kubeconfig path.
// If kubeconfig is empty, it defaults to ~/.kube/config.
func Execute(kubeconfig string, args ...string) (string, error) {
	if kubeconfig == "" {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("could not get current user: %w", err)
		}
		kubeconfig = filepath.Join(usr.HomeDir, ".kube", "config")
	}

	cmd := exec.Command("helm", args...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	output, err := cmd.CombinedOutput() // get stdout and stderr
	return string(output), err
}

// InstallChartFromPath installs a Helm chart from a local directory using Rancher kubeconfig
func InstallChartFromPath(chartName, chartPath, chartVersion, namespace string) error {
	fullChartPath := filepath.Join(chartPath, chartVersion, "/.")

	if _, err := os.Stat(fullChartPath); os.IsNotExist(err) {
		return fmt.Errorf("chart path does not exist: %s", fullChartPath)
	}
	// Helm install arguments
	args := []string{
		"install", chartName, fullChartPath,
		"-n", namespace, "--create-namespace",
	}

	// Run the Helm command
	if _, err := Execute("", args...); err != nil {
		return fmt.Errorf("helm install failed for %s: %w", chartName, err)
	}

	log.Printf("Successfully installed chart '%s' from '%s' into namespace '%s'\n", chartName, fullChartPath, namespace)
	return nil
}
