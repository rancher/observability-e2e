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
	"fmt"
	"net/http"
	"strings"

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

	It("Test : Verify SCC registration status for unregistered Prime cluster", Label("LEVEL0", "SCC", "E2E"), func() {
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
})
