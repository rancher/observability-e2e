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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/rancher/tests/v2/actions/charts"
	"github.com/rancher/rancher/tests/v2/actions/registries"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	extencharts "github.com/rancher/shepherd/extensions/charts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const exampleAppProjectName = "demo-project"

var _ = Describe("Observability Installation Test Suite", func() {
	var clientWithSession *rancher.Client
	var err error

	BeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Should install monitoring chart if not already installed", func() {
		By("Checking if the monitoring chart is already installed")
		initialMonitoringChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, charts.RancherMonitoringName)
		Expect(err).NotTo(HaveOccurred())

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

			By("Installing monitoring chart with the latest version")
			err = charts.InstallRancherMonitoringChart(clientWithSession, monitoringInstOpts, monitoringOpts)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for monitoring chart deployments to have expected replicas")
			err = extencharts.WatchAndWaitDeployments(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for monitoring chart DaemonSets to have expected nodes")
			err = extencharts.WatchAndWaitDaemonSets(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for monitoring chart StatefulSets to have expected replicas")
			err = extencharts.WatchAndWaitStatefulSets(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
		}

		By("Checking if the correct registry prefix is used")
		isUsingRegistry, err := registries.CheckAllClusterPodsForRegistryPrefix(clientWithSession, cluster.ID, registrySetting.Value)
		Expect(err).NotTo(HaveOccurred())
		Expect(isUsingRegistry).To(BeTrue(), "Checking if using correct registry prefix")
	})

	It("Should install Alerting chart if not already installed", func() {
		alertingChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherAlertingNamespace, charts.RancherAlertingName)
		Expect(err).NotTo(HaveOccurred())

		if !alertingChart.IsAlreadyInstalled {
			// Get latest versions of alerting
			latestAlertingVersion, err := clientWithSession.Catalog.GetLatestChartVersion(charts.RancherAlertingName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())

			alertingChartInstallOption := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestAlertingVersion,
				ProjectID: project.ID,
			}

			alertingFeatureOption := &charts.RancherAlertingOpts{
				SMS:   true,
				Teams: true,
			}

			By("Installing alerting chart with the latest version")
			err = charts.InstallRancherAlertingChart(clientWithSession, alertingChartInstallOption, alertingFeatureOption)
			Expect(err).NotTo(HaveOccurred())
		}

		By("Checking if the correct registry prefix is used")
		isUsingRegistry, err := registries.CheckAllClusterPodsForRegistryPrefix(clientWithSession, cluster.ID, registrySetting.Value)
		Expect(err).NotTo(HaveOccurred())
		Expect(isUsingRegistry).To(BeTrue(), "Checking if using correct registry prefix")
	})

	It("Should install Logging chart if not already installed", func() {
		loggingChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherLoggingNamespace, charts.RancherLoggingName)
		Expect(err).NotTo(HaveOccurred())

		if !loggingChart.IsAlreadyInstalled {
			// Get latest versions of logging
			latestLoggingVersion, err := clientWithSession.Catalog.GetLatestChartVersion(charts.RancherLoggingName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())

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
			Expect(err).NotTo(HaveOccurred())
		}

		By("Checking if the correct registry prefix is used")
		isUsingRegistry, err := registries.CheckAllClusterPodsForRegistryPrefix(clientWithSession, cluster.ID, registrySetting.Value)
		Expect(err).NotTo(HaveOccurred())
		Expect(isUsingRegistry).To(BeTrue(), "Checking if using correct registry prefix")
	})
})

// func TestGinkgoSuite(t *testing.T) {
// 	RegisterFailHandler(Fail)
// 	RunSpecs(t, "Installation Test Suite")
// }
