package e2e_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/observability-e2e/tests/helper/utils"
	"github.com/rancher/rancher/tests/v2/actions/namespaces"
	rancher "github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/kubectl"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("Observability Prometheus Federator e2e Test Suite", func() {
	var clientWithSession *rancher.Client

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Test : Verify status of rancher prometheus-federator (deployment + pod) using kubectl", Label("LEVEL0", "promfed", "E2E"), func() {
		By("Step 1) Checking the 'prometheus-federator' deployment in cattle-monitoring-system")
		fetchDeployment := []string{
			"kubectl", "get", "deploy", "prometheus-federator", "-n", "cattle-monitoring-system", "--no-headers",
		}
		deployOutput, err := kubectl.Command(clientWithSession, nil, "local", fetchDeployment, "")
		if err != nil {
			e2e.Failf("Failed to get the prometheus-federator deployment. Error: %v", err)
		}
		Expect(deployOutput).To(MatchRegexp("prometheus-federator.*1/1"), fmt.Sprintf("Expected 'prometheus-federator' deployment to be 1/1, but got: %s", deployOutput))

		By("Step 2) Checking the 'prometheus-federator' pod in cattle-monitoring-system")
		fetchPods := []string{
			"kubectl", "get", "pods", "-n", "cattle-monitoring-system", "-l", "release=prometheus-federator",
		}
		podsOutput, err := kubectl.Command(clientWithSession, nil, "local", fetchPods, "")
		if err != nil {
			e2e.Failf("Failed to get pods in 'cattle-monitoring-system'. Error: %v", err)
		}
		Expect(podsOutput).To(MatchRegexp("prometheus.*1/1.*Running"), fmt.Sprintf("Expected 'prometheus-federator' pod to be running, but got: %s", podsOutput))
	})

	It("Test : Project Monitoring for test-promfed-monitoring", Label("LEVEL0", "promfed", "E2E", "Fedtest"), func() {
		e2e.Logf("Creating new project for Project Monitoring")
		projectConfig := &management.Project{
			ClusterID: cluster.ID,
			Name:      "test-promfed-monitoring",
		}

		project, err = client.Management.Project.Create(projectConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(project.Name).To(Equal("test-promfed-monitoring"))

		e2e.Logf("Creating namespace to deploy Project Monitoring")
		namespace, err := namespaces.CreateNamespace(client, "promfed-monitoring-test-ns", "{}", map[string]string{}, map[string]string{}, project)
		Expect(err).NotTo(HaveOccurred())
		Expect(namespace.Name).To(Equal("promfed-monitoring-test-ns"))
		resourceNamespace := "cattle-project-" + strings.TrimPrefix(project.ID, "local:")

		By("Deploying Project Monitoring chart in the newly created project")
		defer func() {
			depMonResError := utils.DeleteYamlResource(clientWithSession, "../helper/yamls/projectMonitoringChart.yaml", resourceNamespace)
			Expect(depMonResError).To(BeNil(), "Failed to delete Project Monitoring resource")
		}()
		depMonResError := utils.DeployYamlResource(clientWithSession, "../helper/yamls/projectMonitoringChart.yaml", resourceNamespace)
		Expect(depMonResError).To(BeNil(), "Failed to deploy Project Monitoring resource")

		By("Check Project Monitoring resource is Deployed")
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		err = wait.PollUntilContextTimeout(ctx, 10*time.Second, 90*time.Second, true, func(ctx context.Context) (bool, error) {
			getResource := []string{"kubectl", "get", "ProjectHelmChart", "-n", resourceNamespace, "project-monitoring", "-o", "jsonpath={.status.status}"}
			resourceOutput, cmdErr := kubectl.Command(clientWithSession, nil, cluster.Name, getResource, "")
			if cmdErr != nil {
				return false, cmdErr
			}

			if strings.Contains(resourceOutput, "Deployed") {
				e2e.Logf("Project Monitoring Resource is: %s", resourceOutput)
				return true, nil
			}
			e2e.Logf("Project Monitoring resource is not yet Deployed, retrying...")
			return false, nil
		},
		)
		Expect(err).To(BeNil(), "Project Monitoring resource failed to reach 'Deployed' status")
		e2e.Logf("Project Monitoring deployed successfully")
	})
})
