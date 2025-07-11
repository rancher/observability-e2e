package rancher

import (
	"fmt"
	"strings"
	"time"

	e2e "k8s.io/kubernetes/test/e2e/framework"

	"github.com/rancher/observability-e2e/tests/helper/helm"
)

const (
	rancherNamespace = "cattle-system"
	releaseName      = "rancher"
)

// AddRancherHelmRepo adds the Rancher Helm repository and updates it.
func AddRancherHelmRepo(kubeconfig, helmRepoURL, repoName string) error {
	e2e.Logf("Adding Helm repo: %s -> %s", repoName, helmRepoURL)
	if _, err := helm.Execute(kubeconfig, "repo", "add", repoName, helmRepoURL); err != nil {
		return fmt.Errorf("failed to add helm repo: %w", err)
	}

	e2e.Logf("Updating Helm repos...")
	if _, err := helm.Execute(kubeconfig, "repo", "update"); err != nil {
		return fmt.Errorf("failed to update helm repos: %w", err)
	}

	e2e.Logf("Helm repo added and updated successfully")
	return nil
}

// deployRancher is a shared helper for install or upgrade.
func deployRancher(kubeconfig, helmRepoURL, rancherVersion, hostname, password, action string) error {
	repoName := fmt.Sprintf("rancher-%d", time.Now().Unix())
	if err := AddRancherHelmRepo(kubeconfig, helmRepoURL, repoName); err != nil {
		return err
	}

	chart := fmt.Sprintf("%s/rancher", repoName)
	version := strings.TrimPrefix(rancherVersion, "v")

	args := []string{}
	if action == "install" {
		args = append(args, "install", releaseName)
	} else {
		args = append(args, "upgrade", "--install", releaseName)
	}
	args = append(args,
		chart,
		"--namespace", rancherNamespace,
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
	)

	if strings.Contains(helmRepoURL, "releases.rancher.com") {
		e2e.Logf("Rancher using official release chart...")
	} else {
		e2e.Logf("Rancher using SUSE private registry chart...")
		extraArgs := []string{
			"--set", fmt.Sprintf("rancherImageTag=%s", rancherVersion),
			"--set", "rancherImage=stgregistry.suse.com/rancher/rancher",
			"--set", "rancherImagePullPolicy=Always",
			"--set", "extraEnv[0].name=CATTLE_AGENT_IMAGE",
			"--set", fmt.Sprintf("extraEnv[0].value=stgregistry.suse.com/rancher/rancher-agent:%s", rancherVersion),
		}
		args = append(args, extraArgs...)
	}

	output, err := helm.Execute(kubeconfig, args...)
	if err != nil {
		return fmt.Errorf("helm %s failed: %w\nOutput: %s", action, err, output)
	}

	e2e.Logf("Rancher %sed successfully:\n%s", action, output)
	return nil
}

// InstallRancher installs Rancher
func InstallRancher(kubeconfig, helmRepoURL, rancherVersion, hostname, password string) error {
	return deployRancher(kubeconfig, helmRepoURL, rancherVersion, hostname, password, "install")
}

// UpgradeRancher upgrades Rancher
func UpgradeRancher(kubeconfig, helmRepoURL, rancherVersion, hostname, password string) error {
	return deployRancher(kubeconfig, helmRepoURL, rancherVersion, hostname, password, "upgrade")
}
