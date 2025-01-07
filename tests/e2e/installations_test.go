/*
Copyright © 2023 - 2024 SUSE LLC

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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/observability-e2e/tests/helper/charts"
	"github.com/rancher/observability-e2e/tests/helper/utils"
	rancher "github.com/rancher/shepherd/clients/rancher"
	catalog "github.com/rancher/shepherd/clients/rancher/catalog"
	extencharts "github.com/rancher/shepherd/extensions/charts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	exampleAppProjectName  = "System"
	syslogResourceYamlPath = "../helper/yamls/syslogResources.yaml"
)

var _ = Describe("Observability Installation Test Suite", func() {
	var clientWithSession *rancher.Client
	var err error

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Install monitoring chart", Label("LEVEL0", "monitoring", "installation"), func() {
		By("Checking if the monitoring chart is already installed")
		initialMonitoringChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, charts.RancherMonitoringName)
		Expect(err).NotTo(HaveOccurred())
		if initialMonitoringChart.IsAlreadyInstalled {
			e2e.Logf("Monitoring chart is already installated in project: %v", exampleAppProjectName)
		}

		if !initialMonitoringChart.IsAlreadyInstalled {
			// Get latest versions of monitoring
			latestMonitoringVersion, err := clientWithSession.Catalog.GetLatestChartVersion(charts.RancherMonitoringName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())

			monitoringInstOpts := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestMonitoringVersion,
				ProjectID: project.ID,
			}

			monitoringOpts := &charts.RancherMonitoringOpts{
				IngressNginx:      true,
				ControllerManager: true,
				Etcd:              true,
				Proxy:             true,
				Scheduler:         true,
			}
			e2e.Logf("Retrieved latest monitoring chart version to install: %v", latestMonitoringVersion)

			By("Installing monitoring chart with the latest version")
			err = charts.InstallRancherMonitoringChart(clientWithSession, monitoringInstOpts, monitoringOpts)
			if err != nil {
				e2e.Failf("Failed to install the monitoring chart. Error: %v", err)
			}

			By("Waiting for monitoring chart deployments to have expected replicas")
			errDeployChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitDeployments(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
				errDeployChan <- err
			}()

			select {
			case err := <-errDeployChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitDeployments to complete")
			}

			By("Waiting for monitoring chart DaemonSets to have expected nodes")
			errDaemonChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitDaemonSets(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
				errDaemonChan <- err
			}()

			select {
			case err := <-errDaemonChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitDaemonSets to complete")
			}

			By("Waiting for monitoring chart StatefulSets to have expected replicas")
			errStsChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitStatefulSets(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
				errStsChan <- err
			}()

			select {
			case err := <-errStsChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitStatefulSets to complete")
			}
		}
	})

	It("Install Alerting chart", Label("LEVEL0", "alerting", "installation"), func() {
		alertingChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherAlertingNamespace, charts.RancherAlertingName)
		Expect(err).NotTo(HaveOccurred())

		if !alertingChart.IsAlreadyInstalled {
			// Get latest versions of alerting
			latestAlertingVersion, err := clientWithSession.Catalog.GetLatestChartVersion(charts.RancherAlertingName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())
			e2e.Logf("Retrieved latest alerting chart version to install: %v", latestAlertingVersion)

			alertingChartInstallOption := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestAlertingVersion,
				ProjectID: project.ID,
			}

			alertingFeatureOption := &charts.RancherAlertingOpts{
				SMS:   true,
				Teams: false,
			}

			By("Installing alerting chart with the latest version")
			err = charts.InstallRancherAlertingChart(clientWithSession, alertingChartInstallOption, alertingFeatureOption)
			if err != nil {
				e2e.Failf("Failed to install the alerting chart. Error: %v", err)
			}

			By("Waiting for alerting chart deployments to have expected replicas")
			errDeployChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitDeployments(clientWithSession, project.ClusterID, charts.RancherAlertingNamespace, metav1.ListOptions{})
				errDeployChan <- err
			}()

			select {
			case err := <-errDeployChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitDeployments to complete")
			}

			By("Waiting for alerting chart StatefulSets to have expected replicas")
			errStsChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitStatefulSets(clientWithSession, project.ClusterID, charts.RancherAlertingNamespace, metav1.ListOptions{})
				errStsChan <- err
			}()

			select {
			case err := <-errStsChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitStatefulSets to complete")
			}
		}
	})

	It("Install Logging chart", Label("LEVEL0", "logging", "installation"), func() {
		loggingChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherLoggingNamespace, charts.RancherLoggingName)
		Expect(err).NotTo(HaveOccurred())

		if !loggingChart.IsAlreadyInstalled {
			// Get latest versions of logging
			latestLoggingVersion, err := clientWithSession.Catalog.GetLatestChartVersion(charts.RancherLoggingName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())
			e2e.Logf("Retrieved latest logging chart version to install: %v", latestLoggingVersion)

			loggingChartInstallOption := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestLoggingVersion,
				ProjectID: project.ID,
			}

			loggingChartFeatureOption := &charts.RancherLoggingOpts{
				AdditionalLoggingSources: true,
			}

			By("Installing logging chart with the latest version")
			err = charts.InstallRancherLoggingChart(clientWithSession, loggingChartInstallOption, loggingChartFeatureOption)
			if err != nil {
				e2e.Failf("Failed to install the logging chart. Error: %v", err)
			}

			By("Waiting for logging chart deployments to have expected replicas")
			errDeployChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitDeployments(clientWithSession, project.ClusterID, charts.RancherLoggingNamespace, metav1.ListOptions{})
				errDeployChan <- err
			}()

			select {
			case err := <-errDeployChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitDeployments to complete")
			}

			By("Waiting for logging chart DaemonSets to have expected nodes")
			errDaemonChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitDaemonSets(clientWithSession, project.ClusterID, charts.RancherLoggingNamespace, metav1.ListOptions{})
				errDaemonChan <- err
			}()

			select {
			case err := <-errDaemonChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitDaemonSets to complete")
			}

			By("Waiting for logging chart StatefulSets to have expected replicas")
			errStsChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitStatefulSets(clientWithSession, project.ClusterID, charts.RancherLoggingNamespace, metav1.ListOptions{})
				errStsChan <- err
			}()

			select {
			case err := <-errStsChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitStatefulSets to complete")
			}
		}
	})

	It("Install Syslog resources to capture rancher logging logs", Label("LEVEL0", "Syslog", "installation"), func() {

		By("1) Deploying syslog deployment/service/config map resources")
		deploySyslogError := utils.DeploySyslogResources(clientWithSession, syslogResourceYamlPath)
		if deploySyslogError != nil {
			e2e.Failf("Failed to deploy syslog resources: %v", deploySyslogError)
		} else {
			e2e.Logf("Syslog resources deployed successfully!")
		}

	})

	It("Install Prometheus Federator chart", Label("LEVEL0", "promfed", "installation"), func() {

		By("1) verify if prometheus federator chart is already installed")
		prometheusFederatorChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.PrometheusFederatorNamespace, charts.PrometheusFederatorName)
		Expect(err).NotTo(HaveOccurred())

		if !prometheusFederatorChart.IsAlreadyInstalled {
			// Get latest versions of porm-fed chart
			By("2) Fetch latest version of prometheus federator chart")
			latestPrometheusFederatorVersion, err := clientWithSession.Catalog.GetLatestChartVersion(charts.PrometheusFederatorName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())
			e2e.Logf("Retrieved latest promethues-federator chart version to install: %v", latestPrometheusFederatorVersion)

			prometheusFederatorChartInstallOption := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestPrometheusFederatorVersion,
				ProjectID: project.ID,
			}

			prometheusFeatureOption := &charts.PrometheusFederatorOpts{
				EnablePodSecurity: false,
			}

			By("3) Installing prometheus federator chart with the latest version")
			err = charts.InstallPrometheusFederatorChart(clientWithSession, prometheusFederatorChartInstallOption, prometheusFeatureOption)
			if err != nil {
				e2e.Failf("Failed to install the prometheus chart. Error: %v", err)
			} else {
				e2e.Logf("Result | Prometheus Federator chart installed successfully")
			}
		} else {
			e2e.Logf("Result | Prometheus Federator chart is already installed in project: %v", prometheusFederatorChart)
		}
	})

})
