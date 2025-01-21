package e2e_test

import (
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

const (
	loggingResourceYamlPath = "../helper/yamls/clusterOutputandClusterFlow.yaml"
)

var _ = Describe("Observability Logging E2E Test Suite", func() {
	var clientWithSession *rancher.Client // RancherConfig *Config

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Test : Verify status of rancher-logging Deployments using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the deployments belonging to rancher-logging")
		fetchDeployments := []string{"kubectl", "get", "deployments", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingDeployments, err := kubectl.Command(clientWithSession, nil, "local", fetchDeployments, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get deployments.")

		By("1) Read all the deployments and verify the status of rancher-logging deployments")
		foundRancherLogging := false
		deployments := strings.Split(rancherLoggingDeployments, "\n")
		for _, deployment := range deployments {
			if deployment == "" {
				continue
			}

			fields := strings.Fields(deployment)
			Expect(len(fields)).To(BeNumerically(">=", 4), "Unexpected output format for deployment: %s", deployment)

			deploymentName := fields[0]
			readyReplicas := fields[1]
			availableReplicas := fields[3]

			readyCount := strings.Split(readyReplicas, "/")[0]
			desiredCount := strings.Split(readyReplicas, "/")[1]

			if strings.HasPrefix(deploymentName, "rancher-logging") {
				foundRancherLogging = true

				Expect(availableReplicas).To(Equal(desiredCount), "Failure: Deployment %s is not fully available. Desired: %s, Available: %s", deploymentName, desiredCount, availableReplicas)
				Expect(readyCount).To(Equal(desiredCount), "Failure: Deployment %s pods are not fully ready. Desired: %s, Ready: %s", deploymentName, desiredCount, readyCount)
			}
		}
		Expect(foundRancherLogging).To(BeTrue(), "No deployments found starting with 'rancher-logging'")
	})

	It("Test : Verify status of rancher-logging pods using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the pods belongs to rancher-logging")
		fetchPods := []string{"kubectl", "get", "pods", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingPods, err := kubectl.Command(clientWithSession, nil, "local", fetchPods, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get pods.")

		By("1) Read all the pods and verify the status of rancher-logging-Pods")
		rancherLoggingPodFound := false
		pods := strings.Split(rancherLoggingPods, "\n")
		for _, pod := range pods {
			if pod == "" {
				continue
			}
			fields := strings.Fields(pod)
			Expect(len(fields)).To(BeNumerically(">=", 3), "Unexpected output format for pod: %s", pod)

			podName := fields[0]
			podStatus := fields[2]

			if strings.HasPrefix(podName, "rancher-logging") && (podStatus == "Running" || podStatus == "Completed") {
				rancherLoggingPodFound = true
			}
		}
		Expect(rancherLoggingPodFound).To(BeTrue(), "Pod with name 'rancher-logging' is not running or not present")
	})

	It("Test : Verify status of rancher-logging DaemonSets using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the daemon sets belongs to rancher-logging")
		fetchDaemonSets := []string{"kubectl", "get", "daemonsets", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingDaemonSets, err := kubectl.Command(clientWithSession, nil, "local", fetchDaemonSets, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get daemonsets.")

		By("1) Read all the daemonSet and verify the status of rancher-logging-daemonSets")
		daemonSets := strings.Split(rancherLoggingDaemonSets, "\n")
		for _, daemonSet := range daemonSets {
			if daemonSet == "" {
				continue
			}

			fields := strings.Fields(daemonSet)
			Expect(len(fields)).To(BeNumerically(">=", 6), "Unexpected output format for daemonSet: %s", daemonSet)

			daemonSetName := fields[0]
			desiredPods := fields[1]
			readyPods := fields[3]
			availablePods := fields[5]

			Expect(availablePods).To(Equal(desiredPods), "DaemonSet %s is not fully available. Desired: %s, Available: %s", daemonSetName, desiredPods, availablePods)
			Expect(readyPods).To(Equal(desiredPods), "DaemonSet %s is not fully ready. Desired: %s, Ready: %s", daemonSetName, desiredPods, readyPods)
		}
	})

	It("Test : Verify status of rancher-logging StatefulSets using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the StatefulSets belongs to rancher-logging")
		fetchStatefulsets := []string{"kubectl", "get", "statefulsets", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingStatefulsets, err := kubectl.Command(clientWithSession, nil, "local", fetchStatefulsets, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get statefulsets.")

		By("1) Read all the statefulsets and verify the status of rancher-logging-statefulsets")
		statefulsets := strings.Split(rancherLoggingStatefulsets, "\n")
		for _, statefulset := range statefulsets {
			if statefulset == "" {
				continue
			}

			fields := strings.Fields(statefulset)
			Expect(len(fields)).To(BeNumerically(">=", 3), "Unexpected output format for statefulSet: %s", statefulset)

			statefulsetsName := fields[0]
			readyReplicas := fields[1]

			readyCount := strings.Split(readyReplicas, "/")[0]
			desiredCount := strings.Split(readyReplicas, "/")[1]

			if strings.HasPrefix(statefulsetsName, "rancher-logging") {
				Expect(readyCount).To(Equal(desiredCount), "Failure: Deployment %s pods are not fully ready. Desired: %s, Ready: %s", statefulsetsName, desiredCount, readyCount)
			}
		}
	})

	It("Test: Verify creation of Rancher cluster output and cluster flow", Label("LEVEL1", "Logging", "E2E"), func() {

		By("1) Fetching syslog service IP for cluster output host")
		syslogServiceCmd := []string{"kubectl", "get", "svc", "syslog-ng-service", "-n", "cattle-logging-system", "--no-headers"}
		syslogServiceOutput, err := kubectl.Command(clientWithSession, nil, "local", syslogServiceCmd, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to get syslog service")

		serviceFields := strings.Fields(syslogServiceOutput)
		var serviceIP string
		Expect(len(serviceFields)).To(BeNumerically(">", 2), "Failed to extract service IP from syslog service output")
		serviceIP = serviceFields[2]
		e2e.Logf("Syslog service IP: %s", serviceIP)

		By("2) Updating cluster flow and output manifest file")
		yamlContent, err := os.ReadFile(loggingResourceYamlPath)
		Expect(err).NotTo(HaveOccurred(), "Error reading YAML file")

		yamlStr := string(yamlContent)
		updatedYamlStr := strings.Replace(yamlStr, "<syslog service IP>", serviceIP, -1)

		err = os.WriteFile(loggingResourceYamlPath, []byte(updatedYamlStr), 0644)
		Expect(err).NotTo(HaveOccurred(), "Error writing updated YAML file")

		By("3) Deploying cluster output and cluster flow")
		deployLoggingResourcesError := utils.DeployLoggingClusterOutputAndClusterFlow(clientWithSession, loggingResourceYamlPath)
		Expect(deployLoggingResourcesError).NotTo(HaveOccurred(), "Failed to deploy cluster output and flow")

		By("4) Fetching cluster output")
		clusterOutputCmd := []string{"kubectl", "get", "clusteroutput", "testclusteroutput", "-n", "cattle-logging-system"}
		clusterOutput, err := kubectl.Command(clientWithSession, nil, "local", clusterOutputCmd, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to fetch cluster output")
		Expect(clusterOutput).NotTo(BeEmpty(), "Cluster output is empty")

		By("5) Fetching cluster flow")
		clusterFlowCmd := []string{"kubectl", "get", "clusterflow", "testclusterflow", "-n", "cattle-logging-system"}
		clusterFlow, err := kubectl.Command(clientWithSession, nil, "local", clusterFlowCmd, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to fetch cluster flow")
		Expect(clusterFlow).NotTo(BeEmpty(), "Cluster flow is empty")

		By("6) Scaling Rancher Logging Deployment")
		scaleRancherLogging := []string{"kubectl", "scale", "deployment", "rancher-logging", "-n", "cattle-logging-system", "--replicas=4"}
		fetchRancherLogging := []string{"kubectl", "get", "pods", "-n", "cattle-logging-system"}
		scaleDeploymentOutput, err := kubectl.Command(clientWithSession, nil, "local", scaleRancherLogging, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to scale Rancher logging deployment")
		e2e.Logf("Successfully scaled Rancher logging deployment:\n %v", scaleDeploymentOutput)

		fetchRancherLoggingOutput, err := kubectl.Command(clientWithSession, nil, "local", fetchRancherLogging, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to fetch Rancher logging pods")
		e2e.Logf("Successfully fetched Rancher logging pods:\n %v", fetchRancherLoggingOutput)

		var maxRetries = 5
		var retryInterval = 60 * time.Second
		var attempt = 0

		By("7) Verifying Rancher logs via syslog")
		time.Sleep(30 * time.Second)
		syslogPodsCmd := []string{"kubectl", "get", "pods", "-n", "cattle-logging-system", "--no-headers"}
		syslogPodsOutput, err := kubectl.Command(clientWithSession, nil, "local", syslogPodsCmd, "")
		Expect(err).NotTo(HaveOccurred(), "Failed to fetch syslog pods")

		syslogPods := strings.Split(syslogPodsOutput, "\n")
		Expect(syslogPods).NotTo(BeEmpty(), "No syslog pods found")

		for _, pod := range syslogPods {
			if strings.Contains(pod, "syslog-ng-deployment") {
				podFields := strings.Fields(pod)
				Expect(len(podFields)).To(BeNumerically(">", 0), "Invalid pod format")

				podName := podFields[0]
				e2e.Logf("Fetching logs for syslog pod: %v", podName)
				var syslogLogsOutput string
				var err error

				for attempt < maxRetries {
					syslogLogsCmd := []string{"kubectl", "logs", podName, "-n", "cattle-logging-system"}
					syslogLogsOutput, err = kubectl.Command(clientWithSession, nil, "local", syslogLogsCmd, "")
					Expect(err).NotTo(HaveOccurred(), "Error fetching logs for pod %s", podName)

					if syslogLogsOutput == "" || syslogLogsOutput == "null" {
						e2e.Logf("Logs are empty or null. Retrying... (Attempt %d/%d)\n", attempt+1, maxRetries)
						attempt++
						if attempt < maxRetries {
							time.Sleep(retryInterval)
						}
					} else {
						logLines := strings.Split(syslogLogsOutput, "\n")
						startIdx := max(0, len(logLines)-5)
						e2e.Logf("Syslog found:\n%s", strings.Join(logLines[startIdx:], "\n"))
						break
					}
				}

				Expect(attempt).Should(BeNumerically("<", maxRetries), "Failed to fetch valid logs for pod after max retries")
				Expect(syslogLogsOutput).To(
					Or(
						ContainSubstring("testclusteroutput"), ContainSubstring("cattle-logging-system"), ContainSubstring("syslog-ng"),
					),
					"Logs for pod %s did not contain any of the expected substrings: 'testclusteroutput', 'cattle-logging-system', or 'cattle-monitoring-system'", podName)

				attempt = 0
			}
		}

		resetOriginalYaml := strings.Replace(yamlStr, serviceIP, "<syslog service IP>", -1)
		err = os.WriteFile(loggingResourceYamlPath, []byte(resetOriginalYaml), 0644)
		Expect(err).NotTo(HaveOccurred(), "Error writing reset YAML file")
	})

})
