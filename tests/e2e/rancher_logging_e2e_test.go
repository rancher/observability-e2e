package e2e_test

import (
	"fmt"
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
	var clientWithSession *rancher.Client //RancherConfig *Config

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Test : Verify status of rancher-logging Deployments using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the deployments belonging to rancher-logging")
		fetchDeployments := []string{"kubectl", "get", "deployments", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingDeployments, err := kubectl.Command(clientWithSession, nil, "local", fetchDeployments, "")
		if err != nil {
			e2e.Failf("Failed to get deployments. Error: %v", err)
		}

		By("1) Read all the deployments and verify the status of rancher-logging deployments")
		foundRancherLogging := false
		deployments := strings.Split(rancherLoggingDeployments, "\n")
		for _, deployment := range deployments {
			if deployment == "" {
				continue
			}

			fields := strings.Fields(deployment) // Split the line into deployment name and its current status
			if len(fields) < 4 {
				e2e.Failf("Unexpected output format for deployment: %s", deployment)
			}

			deploymentName := fields[0]
			readyReplicas := fields[1]
			availableReplicas := fields[3]

			readyCount := strings.Split(readyReplicas, "/")[0]
			desiredCount := strings.Split(readyReplicas, "/")[1]

			if strings.HasPrefix(deploymentName, "rancher-logging") {
				foundRancherLogging = true

				if availableReplicas == desiredCount {
					e2e.Logf("Success: Deployment %s is fully available. Desired: %s, Available: %s", deploymentName, desiredCount, availableReplicas)
				} else {
					e2e.Failf("Failure: Deployment %s is not fully available. Desired: %s, Available: %s", deploymentName, desiredCount, availableReplicas)
				}

				if readyCount == desiredCount {
					e2e.Logf("Success: Deployment %s pods are fully ready. Desired: %s, Ready: %s", deploymentName, desiredCount, readyCount)
				} else {
					e2e.Failf("Failure: Deployment %s pods are not fully ready. Desired: %s, Ready: %s", deploymentName, desiredCount, readyCount)
				}
			}
		}
		if !foundRancherLogging {
			e2e.Failf("No deployments found starting with 'rancher-logging'")
		}

	})

	It("Test : Verify status of rancher-logging pods using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the pods belongs to rancher-logging")
		fetchPods := []string{"kubectl", "get", "pods", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingPods, err := kubectl.Command(clientWithSession, nil, "local", fetchPods, "")
		if err != nil {
			e2e.Failf("Failed to get pods . Error: %v", err)
		}

		By("1) Read all the pods and verify the status of rancher-logging-Pods")
		rancherLoggingPodFound := false
		pods := strings.Split(rancherLoggingPods, "\n")
		for _, pod := range pods {
			if pod == "" {
				continue
			}
			fields := strings.Fields(pod) // Split the line into pod name and its current status
			if len(fields) < 3 {
				e2e.Failf("Unexpected output format for pod: %s", pod)
			}

			podName := fields[0]
			podStatus := fields[2]

			if strings.HasPrefix(podName, "rancher-logging") && (podStatus == "Running" || podStatus == "Completed") {
				rancherLoggingPodFound = true
			}

		}
		if !rancherLoggingPodFound {
			e2e.Failf("Pod with name 'rancher-logging' is not running or not present")
		}

	})

	It("Test : Verify status of rancher-logging DaemonSets using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the daemon sets belongs to rancher-logging")
		fetchDaemonSets := []string{"kubectl", "get", "daemonsets", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingDaemonSets, err := kubectl.Command(clientWithSession, nil, "local", fetchDaemonSets, "")
		if err != nil {
			e2e.Failf("Failed to get daemonsets . Error: %v", err)
		}
		e2e.Logf("Deamonset Fetched --> %v", rancherLoggingDaemonSets)

		By("1) Read all the daemonSet and verify the status of rancher-logging-daemonSets")
		daemonSets := strings.Split(rancherLoggingDaemonSets, "\n")
		for _, daemonSet := range daemonSets {
			if daemonSet == "" {
				continue
			}

			fields := strings.Fields(daemonSet) // Split the line into pod name and its current status
			if len(fields) < 6 {
				e2e.Failf("Unexpected output format for daemonSet: %s", daemonSet)
			}

			daemonSetName := fields[0]
			desiredPods := fields[1]
			readyPods := fields[3]
			availablePods := fields[5]

			if desiredPods != availablePods {
				e2e.Failf("DaemonSet %s is not fully available. Desired: %s, Available: %s", daemonSetName, desiredPods, availablePods)
			}

			if readyPods != desiredPods {
				e2e.Failf("DaemonSet %s is not fully ready. Desired: %s, Ready: %s", daemonSetName, desiredPods, readyPods)
			}
		}

	})

	It("Test : Verify status of rancher-logging StatefulSets using kubectl", Label("LEVEL1", "Logging", "E2E"), func() {

		By("0) Fetch all the StatefulSets belongs to rancher-logging")
		fetchStatefulsets := []string{"kubectl", "get", "statefulsets", "-n", "cattle-logging-system", "--no-headers"}
		rancherLoggingStatefulsets, err := kubectl.Command(clientWithSession, nil, "local", fetchStatefulsets, "")
		if err != nil {
			e2e.Failf("Failed to get daemonsets . Error: %v", err)
		}

		By("1) Read all the statefulsets and verify the status of rancher-logging-statefulsets")
		statefulsets := strings.Split(rancherLoggingStatefulsets, "\n")
		for _, statefulset := range statefulsets {
			if statefulset == "" {
				continue
			}

			fields := strings.Fields(statefulset) // Split the line into pod name and its current status
			if len(fields) < 3 {
				e2e.Failf("Unexpected output format for daemonSet: %s", statefulset)
			}

			statefulsetsName := fields[0]
			readyReplicas := fields[1]

			readyCount := strings.Split(readyReplicas, "/")[0]
			desiredCount := strings.Split(readyReplicas, "/")[1]

			if strings.HasPrefix(statefulsetsName, "rancher-logging") {

				if readyCount == desiredCount {
					e2e.Logf("Success: Deployment %s pods are fully ready. Desired: %s, Ready: %s", statefulsetsName, desiredCount, readyCount)
				} else {
					e2e.Failf("Failure: Deployment %s pods are not fully ready. Desired: %s, Ready: %s", statefulsetsName, desiredCount, readyCount)
				}
			}

		}

	})

	It("Test: Verify creation of Rancher cluster output and cluster flow", Label("LEVEL1", "Logging", "E2E", "test"), func() {

		By("1) Fetching syslog service IP for cluster output host")
		syslogServiceCmd := []string{"kubectl", "get", "svc", "syslog-ng-service", "-n", "cattle-logging-system", "--no-headers"}
		syslogServiceOutput, err := kubectl.Command(clientWithSession, nil, "local", syslogServiceCmd, "")
		if err != nil {
			e2e.Failf("Failed to get syslog service. Error: %v", err)
		}
		serviceFields := strings.Fields(syslogServiceOutput)
		var serviceIP string
		if len(serviceFields) > 2 {
			serviceIP = serviceFields[2]
			e2e.Logf("Syslog service IP: %s", serviceIP)
		} else {
			e2e.Failf("Failed to extract service IP from syslog service output")
		}

		By("2) Updating cluster flow and output manifest file")
		yamlContent, err := os.ReadFile(loggingResourceYamlPath)
		if err != nil {
			e2e.Failf("Error reading YAML file: %v", err)
		}

		yamlStr := string(yamlContent)
		updatedYamlStr := strings.Replace(yamlStr, "<syslog service IP>", serviceIP, -1)

		err = os.WriteFile(loggingResourceYamlPath, []byte(updatedYamlStr), 0644)
		if err != nil {
			e2e.Failf("Error writing updated YAML file: %v", err)
		}

		By("3) Deploying cluster output and cluster flow")
		deployLoggingResourcesError := utils.DeployLoggingClusterOutputAndClusterFlow(clientWithSession, loggingResourceYamlPath)
		if deployLoggingResourcesError != nil {
			e2e.Failf("Failed to deploy cluster output and flow: %v", deployLoggingResourcesError)
		} else {
			e2e.Logf("Cluster output and flow deployed successfully!")
		}

		By("4) Fetching cluster output")
		clusterOutputCmd := []string{"kubectl", "get", "clusteroutput", "testclusteroutput", "-n", "cattle-logging-system"}
		clusterOutput, err := kubectl.Command(clientWithSession, nil, "local", clusterOutputCmd, "")
		if err != nil {
			e2e.Failf("Failed to fetch cluster output. Error: %v", err)
		}
		e2e.Logf("Successfully fetched cluster output:\n %v", clusterOutput)

		By("5) Fetching cluster flow")
		clusterFlowCmd := []string{"kubectl", "get", "clusterflow", "testclusterflow", "-n", "cattle-logging-system"}
		clusterFlow, err := kubectl.Command(clientWithSession, nil, "local", clusterFlowCmd, "")
		if err != nil {
			e2e.Failf("Failed to fetch cluster flow. Error: %v", err)
		}
		e2e.Logf("Successfully fetched cluster flow:\n %v", clusterFlow)

		By("6) Scaling Rancher Logging Deployment")
		scaleRancherLogging := []string{"kubectl", "scale", "deployment", "rancher-logging", "-n", "cattle-logging-system", "--replicas=4"}
		fetchRancherLogging := []string{"kubectl", "get", "pods", "-n", "cattle-logging-system"}
		scaleDeploymentOutput, err := kubectl.Command(clientWithSession, nil, "local", scaleRancherLogging, "")
		if err != nil {
			e2e.Failf("Failed to scale Rancher logging deployment. Error: %v", err)
		}
		e2e.Logf("Successfully scaled Rancher logging deployment:\n %v", scaleDeploymentOutput)

		fetchRancherLoggingOutput, err := kubectl.Command(clientWithSession, nil, "local", fetchRancherLogging, "")
		if err != nil {
			e2e.Failf("Failed to fetch Rancher logging pods. Error: %v", err)
		}
		e2e.Logf("Successfully fetched Rancher logging pods:\n %v", fetchRancherLoggingOutput)

		var maxRetries = 5
		var retryInterval = 60 * time.Second
		var attempt = 0

		By("7) Verifying Rancher logs via syslog")
		time.Sleep(30 * time.Second)
		syslogPodsCmd := []string{"kubectl", "get", "pods", "-n", "cattle-logging-system", "--no-headers"}
		syslogPodsOutput, err := kubectl.Command(clientWithSession, nil, "local", syslogPodsCmd, "")
		if err != nil {
			e2e.Failf("Failed to fetch syslog pods. Error: %v", err)
		}

		syslogPods := strings.Split(syslogPodsOutput, "\n")
		for _, pod := range syslogPods {
			if strings.Contains(pod, "syslog-ng-deployment") {
				podFields := strings.Fields(pod)
				if len(podFields) > 0 {
					podName := podFields[0]
					e2e.Logf("Fetching logs for syslog pod: %v", podName)
					var syslogLogsOutput string
					var err error

					// Retries
					for attempt < maxRetries {
						syslogLogsCmd := []string{"kubectl", "logs", podName, "-n", "cattle-logging-system"}
						syslogLogsOutput, err = kubectl.Command(clientWithSession, nil, "local", syslogLogsCmd, "")
						if err != nil {
							e2e.Failf("Error fetching logs for pod %s: %v", podName, err)
						}

						// If logs are empty or nil, retry
						if syslogLogsOutput == "" || syslogLogsOutput == "null" {
							e2e.Logf("Logs are empty or null. Retrying... (Attempt %d/5)\n", attempt+1)
							attempt++
							if attempt < 5 {
								time.Sleep(retryInterval)
							}
						} else {
							e2e.Logf("Syslog found. %v", syslogLogsOutput)
							break
						}
					}

					if attempt == maxRetries && !strings.Contains(syslogLogsOutput, "testclusteroutput") {
						e2e.Failf("Logs for pod %s did not contain 'testclusteroutput' after %d attempts", podName, maxRetries)
					}
					attempt = 0
				}
			}
		}
		resetOriginalYaml := strings.Replace(yamlStr, serviceIP, "<syslog service IP>", -1)

		err = os.WriteFile(loggingResourceYamlPath, []byte(resetOriginalYaml), 0644)
		if err != nil {
			fmt.Printf("Error writing updated YAML file: %v\n", err)
			return
		}
	})

})
