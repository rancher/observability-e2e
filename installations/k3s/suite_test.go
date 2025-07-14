/*
Copyright © 2025 - 2026 SUSE LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e_test

import (
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
)

// const (
// 	ciTokenYaml         = "./assets/local-kubeconfig-token-skel.yaml"
// 	localKubeconfigYaml = "./assets/local-kubeconfig-skel.yaml"
// )

var (
	k3sVersion         string
	rancherChannel     string
	rancherHeadVersion string
	hostname           string
	rancherVersion     string
)

/**
 * Execute RunHelmBinaryWithCustomErr within a loop with timeout
 * @param s options to pass to RunHelmBinaryWithCustomErr command
 * @returns Nothing, the function will fail through Ginkgo in case of issue
 */
func RunHelmCmdWithRetry(s ...string) {
	Eventually(func() error {
		return kubectl.RunHelmBinaryWithCustomErr(s...)
	}, tools.SetTimeout(2*time.Minute), 20*time.Second).Should(Not(HaveOccurred()))
}

func FailWithReport(message string, callerSkip ...int) {
	// Ensures the correct line numbers are reported
	Fail(message, callerSkip[0]+1)
}

// go test -timeout 30m -run ^TestE2E$ github.com/rancher/observability-e2e/installations/k3s -v -ginkgo.v
func TestE2E(t *testing.T) {
	RegisterFailHandler(FailWithReport)
	RunSpecs(t, "Observability K3S Installation End-To-End Test Suite")
}

var _ = BeforeSuite(func() {
	// export PASSWORD=rancherpassword
	// export HOSTNAME_NAME=localhost
	hostname = os.Getenv("HOSTNAME_NAME")
	if hostname == "" {
		hostname = os.Getenv("localhost")
	}
	rancherVersion = os.Getenv("RANCHER_VERSION")
	if rancherVersion == "" {
		rancherVersion = "latest/devel/2.11"
	}

	k3sVersion = os.Getenv("K3S_VERSION")
	if k3sVersion == "" {
		k3sVersion = "v1.32.6+k3s1"
	}

	// Extract Rancher Manager channel/version to install
	if rancherVersion != "" {
		// Split rancherVersion and reset it
		s := strings.Split(rancherVersion, "/")
		rancherVersion = ""

		// Get needed information
		rancherChannel = s[0]
		if len(s) > 1 {
			rancherVersion = s[1]
		}
		if len(s) > 2 {
			rancherHeadVersion = s[2]
		}
	}
})
