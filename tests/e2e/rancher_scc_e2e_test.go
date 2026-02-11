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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/observability-e2e/tests/helper/utils"
	rancher "github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/kubectl"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("Observability SCC E2E Test Suite", func() {
	var clientWithSession *rancher.Client

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("[QASE-18052] Test : Verify SCC registration status for unregistered Prime cluster", Label("LEVEL0", "SCC", "E2E"), func() {
		testCaseID = 18052
		By("1) Check if Rancher cluster is Prime by verifying deployment image registry")
		isPrime, imageRegistry := utils.CheckIfRancherIsPrime(clientWithSession)
		e2e.Logf("Rancher is Prime: %v, Image Registry: %s", isPrime, imageRegistry)

		if !isPrime {
			Skip("Skipping SCC test: Rancher cluster is not Prime (using Community edition)")
		}

		By("2) Verify registration CRD exists")
		checkRegistrationCRD := []string{"kubectl", "get", "crd", "registrations.scc.cattle.io"}
		crdOutput, err := kubectl.Command(clientWithSession, nil, "local", checkRegistrationCRD, "")

		// Check if CRD exists - kubectl returns error message in output if not found
		if err != nil || strings.Contains(crdOutput, "NotFound") || strings.Contains(crdOutput, "not found") {
			Skip("Skipping SCC test: Registration CRD not found. SCC feature may not be available in this cluster.")
		}
		Expect(crdOutput).NotTo(BeEmpty(), "Registration CRD not found")
		e2e.Logf("Registration CRD exists: %s", strings.TrimSpace(crdOutput))

		By("3) Verify registration resource list is empty for unregistered cluster")
		getRegistrations := []string{"kubectl", "get", "registrations.scc.cattle.io", "-o", "json"}
		registrationOutput, err := kubectl.Command(clientWithSession, nil, "local", getRegistrations, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get registration resources")
		Expect(registrationOutput).NotTo(BeEmpty(), "Registration output is empty")

		// Verify the output contains empty items list
		Expect(registrationOutput).To(ContainSubstring(`"items": []`), "Expected empty registration items for unregistered cluster")
		e2e.Logf("Registration resources are empty as expected for unregistered cluster")

		By("4) Verify SCC settings page is accessible via dashboard endpoint")
		rancherURL := clientWithSession.RancherConfig.Host
		// Ensure URL has proper scheme
		if !strings.HasPrefix(rancherURL, "http://") && !strings.HasPrefix(rancherURL, "https://") {
			rancherURL = "https://" + rancherURL
		}
		settingsEndpoint := fmt.Sprintf("%s/dashboard/c/local/settings/registration", rancherURL)

		statusCode, err := utils.CheckDashboardEndpoint(settingsEndpoint, clientWithSession)
		Expect(err).NotTo(HaveOccurred(), "Failed to access SCC registration settings page")
		Expect(statusCode).To(Equal(http.StatusOK), "Expected HTTP 200 OK for SCC registration settings page, got %d", statusCode)
		e2e.Logf("SCC registration settings page is accessible at: %s", settingsEndpoint)

		By("5) Verify unregistered status")
		e2e.Logf("Successfully verified: Rancher Prime cluster shows unregistered SCC status")
	})

	It("[QASE-18061] Test : Register SCC and verify registration status", Label("LEVEL0", "SCC", "E2E"), func() {
		testCaseID = 18061
		By("1) Check if Rancher cluster is Prime")
		isPrime, imageRegistry := utils.CheckIfRancherIsPrime(clientWithSession)
		e2e.Logf("Rancher is Prime: %v, Image Registry: %s", isPrime, imageRegistry)

		if !isPrime {
			Skip("Skipping SCC test: Rancher cluster is not Prime (using Community edition)")
		}

		By("2) Verify registration CRD exists")
		checkRegistrationCRD := []string{"kubectl", "get", "crd", "registrations.scc.cattle.io"}
		crdOutput, err := kubectl.Command(clientWithSession, nil, "local", checkRegistrationCRD, "")

		if err != nil || strings.Contains(crdOutput, "NotFound") || strings.Contains(crdOutput, "not found") {
			Skip("Skipping SCC test: Registration CRD not found. SCC feature may not be available in this cluster.")
		}

		By("3) Get SCC registration code from environment variable")
		regCode := os.Getenv("SCC_REGCODE")
		if regCode == "" {
			e2e.Logf("SCC_REGCODE environment variable is not set. Please set export SCC_REGCODE=your_registration_code to run SCC test.")
			Skip("Skipping SCC registration test: SCC_REGCODE environment variable is not set")
		}
		e2e.Logf("SCC registration code retrieved from environment variable")

		By("4) Create SCC registration secret")
		createSecret := []string{
			"kubectl", "create", "secret", "generic", "scc-registration",
			"-n", "cattle-scc-system",
			"--from-literal=mode=online",
			fmt.Sprintf("--from-literal=regCode=%s", regCode),
		}
		_, err = kubectl.Command(clientWithSession, nil, "local", createSecret, "")
		if err != nil {
			Fail("Failed to create SCC registration secret")
		}
		e2e.Logf("SCC registration secret created successfully")

		// Register cleanup immediately to ensure it runs even if test fails
		DeferCleanup(func() {
			By("Cleanup: Delete SCC registration secret")
			deleteSecret := []string{"kubectl", "delete", "secret", "scc-registration", "-n", "cattle-scc-system", "--ignore-not-found=true"}
			_, deleteErr := kubectl.Command(clientWithSession, nil, "local", deleteSecret, "")
			if deleteErr != nil {
				e2e.Logf("Warning: Failed to delete SCC registration secret: %v", deleteErr)
			} else {
				e2e.Logf("SCC registration secret deleted successfully")
			}
		})

		By("5) Wait for registration to be processed (30 seconds)")
		time.Sleep(30 * time.Second)

		By("6) Verify registration resource is created")
		getRegistrations := []string{"kubectl", "get", "registrations.scc.cattle.io", "-n", "cattle-scc-system", "-o", "json"}
		registrationOutput, err := kubectl.Command(clientWithSession, nil, "local", getRegistrations, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get registration resources")
		Expect(registrationOutput).NotTo(BeEmpty(), "Registration output is empty")

		// Parse JSON output
		var registrationList map[string]interface{}
		err = json.Unmarshal([]byte(registrationOutput), &registrationList)
		Expect(err).NotTo(HaveOccurred(), "Failed to parse registration JSON output")

		items := registrationList["items"].([]interface{})
		Expect(len(items)).To(BeNumerically(">", 0), "No registration resources found")

		registration := items[0].(map[string]interface{})
		spec := registration["spec"].(map[string]interface{})
		status := registration["status"].(map[string]interface{})

		By("7) Verify registration MODE is online")
		mode := spec["mode"].(string)
		Expect(mode).To(Equal("online"), "Expected MODE to be 'online', got '%s'", mode)
		e2e.Logf("Registration MODE is 'online' as expected")

		By("8) Verify REGISTRATION ACTIVE status")
		activationStatus := status["activationStatus"].(map[string]interface{})
		activated := activationStatus["activated"].(bool)
		Expect(activated).To(BeTrue(), "Expected registration to be activated")
		e2e.Logf("Registration is activated: %v", activated)

		By("9) Verify registered product starts with 'SUSE Rancher'")
		registeredProduct, exists := status["registeredProduct"].(string)
		Expect(exists).To(BeTrue(), "registeredProduct field not found in status")
		Expect(registeredProduct).To(HavePrefix("SUSE Rancher"), "Expected registeredProduct to start with 'SUSE Rancher', got '%s'", registeredProduct)
		e2e.Logf("Registered product: %s", registeredProduct)

		By("10) Verify activation status is true")
		Expect(activationStatus["activated"]).To(BeTrue(), "Expected activationStatus.activated to be true")
		e2e.Logf("Activation status verified successfully")

		By("11) Verify rancher-scc-operator logs show successful registration")
		getOperatorLogs := []string{"kubectl", "logs", "deployments/rancher-scc-operator", "scc-operator", "-n", "cattle-scc-system", "--tail=10"}
		operatorLogs, err := kubectl.Command(clientWithSession, nil, "local", getOperatorLogs, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get rancher-scc-operator logs")
		Expect(operatorLogs).NotTo(BeEmpty(), "Operator logs are empty")

		// Check if logs contain successful registration message
		if strings.Contains(operatorLogs, "Successfully registered activation") {
			e2e.Logf("SUCCESS: Operator logs confirm successful registration activation")
		} else {
			Fail("FAILED: Operator logs do not contain 'Successfully registered activation'. Registration may have failed.")
		}

		e2e.Logf("Successfully verified: SCC registration and activation completed")
	})
})
