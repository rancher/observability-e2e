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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher-sandbox/ele-testhelpers/kubectl"
	"github.com/rancher-sandbox/ele-testhelpers/rancher"
	"github.com/rancher-sandbox/ele-testhelpers/tools"
)

const password = "rancherpassword"

var _ = Describe("E2E - Install Rancher Manager", Label("install"), Ordered, func() {
	k := &kubectl.Kubectl{
		Namespace:    "",
		PollTimeout:  tools.SetTimeout(300 * time.Second),
		PollInterval: 500 * time.Millisecond,
	}
	// Define local Kubeconfig file
	localKubeconfig := os.Getenv("HOME") + "/.kube/config"

	It("Installs Helm CLI", func() {
		By("Downloading the Helm install script", func() {
			cmd := exec.Command("curl", "-fsSL", "-o", "/root/get_helm.sh", "https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3")
			_, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
		})

		By("Making the Helm install script executable", func() {
			cmd := exec.Command("chmod", "+x", "/root/get_helm.sh")
			_, err := cmd.CombinedOutput()
			Expect(err).ToNot(HaveOccurred())
		})

		By("Running the Helm install script", func() {
			cmd := exec.Command("/root/get_helm.sh")
			cmd.Env = os.Environ()
			out, err := cmd.CombinedOutput()
			GinkgoWriter.Printf("helm install script output: %s\n", string(out))
			Expect(err).ToNot(HaveOccurred())
		})

		By("Checking that helm is installed", func() {
			cmd := exec.Command("helm", "version")
			out, err := cmd.CombinedOutput()
			GinkgoWriter.Printf("helm version output: %s\n", string(out))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(out)).To(ContainSubstring("v3"))
		})
	})

	It("Install Rancher Manager", func() {
		By("Installing K3s", func() {
			// Get K3s installation script
			fileName := "k3s-install.sh"
			Eventually(func() error {
				return tools.GetFileFromURL("https://get.k3s.io", fileName, true)
			}, tools.SetTimeout(2*time.Minute), 10*time.Second).ShouldNot(HaveOccurred())

			// Set command and arguments
			installCmd := exec.Command("sh", fileName)
			// Add the INSTALL_K3S_VERSION to the command's environment
			installCmd.Env = append(os.Environ(), "INSTALL_K3S_VERSION="+k3sVersion)

			// Retry in case of (sporadic) failure...
			count := 1
			Eventually(func() error {
				// Execute K3s installation
				out, err := installCmd.CombinedOutput()
				GinkgoWriter.Printf("K3s installation loop %d:\n%s\n", count, out)
				count++
				return err
			}, tools.SetTimeout(2*time.Minute), 5*time.Second).Should(BeNil())
		})

		By("Starting K3s", func() {
			err := exec.Command("sudo", "systemctl", "start", "k3s").Run()
			Expect(err).To(Not(HaveOccurred()))

			// Delay few seconds before checking
			time.Sleep(tools.SetTimeout(20 * time.Second))
		})

		By("Waiting for K3s to be started", func() {
			// Wait for all pods to be started
			checkList := [][]string{
				{"kube-system", "app=local-path-provisioner"},
				{"kube-system", "k8s-app=kube-dns"},
				{"kube-system", "app.kubernetes.io/name=traefik"},
				{"kube-system", "svccontroller.k3s.cattle.io/svcname=traefik"},
			}
			Eventually(func() error {
				return rancher.CheckPod(k, checkList)
			}, tools.SetTimeout(4*time.Minute), 30*time.Second).Should(BeNil())
		})

		By("Configuring Kubeconfig file", func() {
			// Copy K3s file in ~/.kube/config
			// NOTE: don't check for error, as it will happen anyway (only K3s or RKE2 is installed at a time)
			file, _ := exec.Command("bash", "-c", "ls /etc/rancher/{k3s,rke2}/{k3s,rke2}.yaml").Output()
			Expect(file).To(Not(BeEmpty()))
			err := tools.CopyFile(strings.Trim(string(file), "\n"), localKubeconfig)
			Expect(err).To(Not(HaveOccurred()))

			err = os.Setenv("KUBECONFIG", localKubeconfig)
			Expect(err).To(Not(HaveOccurred()))
		})

		By("Installing CertManager", func() {
			RunHelmCmdWithRetry("repo", "add", "jetstack", "https://charts.jetstack.io")
			RunHelmCmdWithRetry("repo", "update")

			// Set flags for cert-manager installation
			flags := []string{
				"upgrade", "--install", "cert-manager", "jetstack/cert-manager",
				"--namespace", "cert-manager",
				"--create-namespace",
				"--set", "installCRDs=true",
				"--wait", "--wait-for-jobs",
			}

			RunHelmCmdWithRetry(flags...)

			checkList := [][]string{
				{"cert-manager", "app.kubernetes.io/component=controller"},
				{"cert-manager", "app.kubernetes.io/component=webhook"},
				{"cert-manager", "app.kubernetes.io/component=cainjector"},
			}
			Eventually(func() error {
				return rancher.CheckPod(k, checkList)
			}, tools.SetTimeout(4*time.Minute), 30*time.Second).Should(BeNil())
		})

		By("Installing Rancher Manager", func() {
			err := rancher.DeployRancherManager(hostname, rancherChannel, rancherVersion, rancherHeadVersion, "none", "none")
			Expect(err).To(Not(HaveOccurred()))

			// Wait for all pods to be started
			checkList := [][]string{
				{"cattle-system", "app=rancher"},
				{"cattle-system", "app=rancher-webhook"},
			}
			Eventually(func() error {
				return rancher.CheckPod(k, checkList)
			}, tools.SetTimeout(4*time.Minute), 30*time.Second).Should(BeNil())
		})

		By("Generating Rancher API token and saving to cattle-config.yaml", func() {
			time.Sleep(60 * time.Second)
			type loginResponse struct {
				Token string `json:"token"`
			}

			var token string
			var err error
			url := "https://localhost/v3-public/localProviders/local?action=login"
			username := "admin"

			for i := 0; i < 3; i++ {
				payload := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
				req, err := http.NewRequest("POST", url, strings.NewReader(payload))
				Expect(err).ToNot(HaveOccurred())
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{
					Timeout: 30 * time.Second,
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
					},
				}

				resp, err := client.Do(req)
				Expect(err).ToNot(HaveOccurred())
				defer resp.Body.Close()

				body, err := io.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())

				var loginResp loginResponse
				err = json.Unmarshal(body, &loginResp)
				Expect(err).ToNot(HaveOccurred())

				if loginResp.Token != "" {
					token = loginResp.Token
					break
				} else {
					GinkgoWriter.Println("Retrying in 20 seconds...")
					time.Sleep(20 * time.Second)
				}
			}

			Expect(token).ToNot(BeEmpty(), "Failed to obtain Rancher API token")

			// Write the token in cattle-config.yaml format
			filePath := os.Getenv("HOME") + "/cattle-config.yaml"
			configContent := fmt.Sprintf(
				"rancher:\n  host: localhost\n  adminToken: %s\n  insecure: True\n  clusterName: local\n  cleanup: true\n",
				token,
			)

			err = os.WriteFile(filePath, []byte(configContent), 0o600)
			Expect(err).ToNot(HaveOccurred())

			GinkgoWriter.Printf("Rancher config saved to %s\n", filePath)
		})
	})
})
