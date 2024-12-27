package e2e_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rancher "github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/kubectl"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("Observability E2E Test Suite", func() {
	var clientWithSession *rancher.Client //RancherConfig *Config

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Test : Verify status of rancher-logging Deployments using kubectl", Label("LEVEL1", "Logging", "E2E", "test"), func() {

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

	It("Test : Verify status of rancher-logging pods using kubectl", Label("LEVEL1", "Logging", "E2E", "test"), func() {

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

	It("Test : Verify status of rancher-logging DaemonSets using kubectl", Label("LEVEL1", "Logging", "E2E", "test"), func() {

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

	It("Test : Verify status of rancher-logging StatefulSets using kubectl", Label("LEVEL1", "Logging", "E2E", "test"), func() {

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

})
