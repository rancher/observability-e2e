/*
Copyright © 2024 - 2025 SUSE LLC

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
	"github.com/rancher/observability-e2e/tests/helper/charts"
	rancher "github.com/rancher/shepherd/clients/rancher"
	catalog "github.com/rancher/shepherd/clients/rancher/catalog"
	extencharts "github.com/rancher/shepherd/extensions/charts"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

var _ = Describe("Observability Upgrade Test Suite", func() {
	var clientWithSession *rancher.Client
	var err error

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("[QASE-3891] Upgrade monitoring chart to the Latest Version", Label("monitoring", "afterUpgrade"), func() {
		testCaseID = 3891
		By("Checking if the monitoring chart is already installed")
		initialMonitoringChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherMonitoringNamespace, charts.RancherMonitoringName)
		Expect(err).NotTo(HaveOccurred())

		if initialMonitoringChart.IsAlreadyInstalled {
			e2e.Logf("Monitoring chart is already installed in project: %v", exampleAppProjectName)
			e2e.Logf("Getting Monitoring Newer Version")
			monitoringVersionChartList, err := clientWithSession.Catalog.GetListChartVersions(charts.RancherMonitoringName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())
			e2e.Logf("Chart List: %v", monitoringVersionChartList)
			if len(monitoringVersionChartList) <= 1 {
				Skip("No newer versions found for the monitoring chart to perform the upgrade")
			}
			latestMonitoringVersion := monitoringVersionChartList[0]

			chartInstallOptions := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestMonitoringVersion,
				ProjectID: project.ID,
			}
			chartFeatureOptions := &charts.RancherMonitoringOpts{
				IngressNginx:      true,
				ControllerManager: true,
				Etcd:              true,
				Proxy:             true,
				Scheduler:         true,
			}
			e2e.Logf("Retrieved latest monitoring chart version to install: %v", latestMonitoringVersion)

			By("Upgrading monitoring chart to the latest version")
			err = charts.UpgradeRancherMonitoringChart(clientWithSession, chartInstallOptions, chartFeatureOptions)
			if err != nil {
				e2e.Failf("Failed to upgrade the monitoring chart. Error: %v", err)
			}
		} else {
			Skip("Monitoring is not installed. Execute the pre-upgrade installation test before attempting the upgrade")
		}
	})

	It("[QASE-8322] Upgrade prometheus federator chart to the Latest Version", Label("promfed", "afterUpgrade"), func() {
		testCaseID = 8322
		By("Checking if the prometheus federator chart is already installed")
		prometheusFederatorChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.PrometheusFederatorNamespace, charts.PrometheusFederatorName)
		Expect(err).NotTo(HaveOccurred())

		if prometheusFederatorChart.IsAlreadyInstalled {
			e2e.Logf("Prometheus federator chart is already installed in project: %v", exampleAppProjectName)
			e2e.Logf("Getting prometheus federator Newer Version")
			promFedChartList, err := clientWithSession.Catalog.GetListChartVersions(charts.PrometheusFederatorName, catalog.RancherChartRepo)

			Expect(err).NotTo(HaveOccurred())
			e2e.Logf("Chart List: %v", promFedChartList)
			if len(promFedChartList) <= 1 {
				Skip("No newer versions found for the prometheus federator chart to perform the upgrade")
			}
			latestMonitoringVersion := promFedChartList[0]

			installOptions := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestMonitoringVersion,
				ProjectID: project.ID,
			}

			prometheusFederatorOpts := &charts.PrometheusFederatorOpts{
				EnablePodSecurity: false,
			}

			e2e.Logf("Retrieved latest prometheus federator chart version to install: %v", latestMonitoringVersion)

			By("Upgrading prometheus federator chart to the latest version")
			err = charts.UpgradePrometheusFederatorChart(clientWithSession, installOptions, prometheusFederatorOpts)
			if err != nil {
				e2e.Failf("Failed to upgrade the prometheus federator chart. Error: %v", err)
			}
		} else {
			Skip("prometheus federator is not installed. Execute the pre-upgrade installation test before attempting the upgrade")
		}
	})
})
