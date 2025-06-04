package rancher

import (
	"fmt"
	"strings"
	"time"

	e2e "k8s.io/kubernetes/test/e2e/framework"

	"github.com/rancher/observability-e2e/tests/helper/helm"
)

// AddRancherHelmRepo adds the Rancher Helm repository and updates it.
func AddRancherHelmRepo(kubeconfig, helmRepoURL, repoName string) error {

	e2e.Logf("Adding Helm repo: %s -> %s", repoName, helmRepoURL)
	_, err := helm.Execute(kubeconfig, "repo", "add", repoName, helmRepoURL)
	if err != nil {
		return fmt.Errorf("failed to add helm repo: %w", err)
	}

	e2e.Logf("Updating Helm repos...")
	_, err = helm.Execute(kubeconfig, "repo", "update")
	if err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	e2e.Logf("Helm repo added and updated successfully")
	return nil
}

// InstallRancher installs Rancher based on the repo URL and version
func InstallRancher(kubeconfig, helmRepoURL, rancherVersion, hostname, password string) error {
	repoName := fmt.Sprintf("rancher-%d", time.Now().Unix())
	if err := AddRancherHelmRepo(kubeconfig, helmRepoURL, repoName); err != nil {
		return err
	}

	namespace := "cattle-system"
	chart := fmt.Sprintf("%s/rancher", repoName)
	version := strings.TrimPrefix(rancherVersion, "v")

	commonArgs := []string{
		"install", "rancher", chart,
		"--namespace", namespace,
		"--version", version,
		"--set", fmt.Sprintf("hostname=%s", hostname),
		"--set", "replicas=2",
		"--set", fmt.Sprintf("bootstrapPassword=%s", password),
		"--set", "global.cattle.psp.enabled=false",
		"--set", "insecure=true",
		"--wait",
		"--timeout=10m",
		"--create-namespace",
		"--devel",
	}

	if strings.Contains(helmRepoURL, "releases.rancher.com") {
		e2e.Logf("Installing Rancher using official release chart...")
	} else {
		e2e.Logf("Installing Rancher using SUSE private registry chart...")
		extraArgs := []string{
			"--set", fmt.Sprintf("rancherImageTag=%s", rancherVersion),
			"--set", "rancherImage=stgregistry.suse.com/rancher/rancher",
			"--set", "rancherImagePullPolicy=Always",
			"--set", "extraEnv[0].name=CATTLE_AGENT_IMAGE",
			"--set", fmt.Sprintf("extraEnv[0].value=stgregistry.suse.com/rancher/rancher-agent:%s", rancherVersion),
		}
		commonArgs = append(commonArgs, extraArgs...)
	}

	output, err := helm.Execute(kubeconfig, commonArgs...)
	if err != nil {
		return fmt.Errorf("helm install failed: %w\nOutput: %s", err, output)
	}

	e2e.Logf("Rancher installed successfully: %s", output)
	return nil
}
