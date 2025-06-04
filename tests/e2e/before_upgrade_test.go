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
	"fmt"

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

	It("Uninstall Existing Observability Charts", Label("beforeUpgrade"), func() {
		chartsToUninstall := []struct {
			name      string
			namespace string
		}{
			{name: charts.RancherMonitoringName, namespace: charts.RancherMonitoringNamespace},
			{name: charts.RancherMonitoringCRDName, namespace: charts.RancherMonitoringNamespace},
			{name: charts.PrometheusFederatorName, namespace: charts.PrometheusFederatorNamespace},
			{name: charts.RancherLoggingName, namespace: charts.RancherLoggingNamespace},
			{name: charts.RancherLoggingCRDName, namespace: charts.RancherLoggingNamespace},
		}

		for _, chartInfo := range chartsToUninstall {
			By(fmt.Sprintf("Checking if the %v chart is already installed", chartInfo.name))
			initialChartStatus, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, chartInfo.namespace, chartInfo.name)
			Expect(err).NotTo(HaveOccurred())

			if initialChartStatus.IsAlreadyInstalled {
				e2e.Logf("Uninstalling chart %v", chartInfo.name)
				err = charts.UninstallChart(clientWithSession, project.ClusterID, chartInfo.name, chartInfo.namespace)
				Expect(err).NotTo(HaveOccurred())
			}
		}
	})

	It("[QASE-3889] Install an older version of the Monitoring Chart", Label("monitoring", "beforeUpgrade"), func() {
		testCaseID = 3889
		e2e.Logf("Getting Monitoring Older Version")
		monitoringVersionChartList, err := clientWithSession.Catalog.GetListChartVersions(charts.RancherMonitoringName, catalog.RancherChartRepo)
		Expect(err).NotTo(HaveOccurred())
		e2e.Logf("Chart List: %v", monitoringVersionChartList)

		if len(monitoringVersionChartList) <= 1 {
			Skip("Not enough older versions found for the monitoring chart to proceed with installation")
		}
		monitoringVersion := monitoringVersionChartList[1]

		monitoringInstOpts := &charts.InstallOptions{
			Cluster:   cluster,
			Version:   monitoringVersion,
			ProjectID: project.ID,
		}

		monitoringOpts := &charts.RancherMonitoringOpts{
			IngressNginx:      true,
			ControllerManager: true,
			Etcd:              true,
			Proxy:             true,
			Scheduler:         true,
		}
		e2e.Logf("Retrieved monitoring chart version to install: %v", monitoringVersion)

		By(fmt.Sprintf("Installing monitoring chart with an %v version", monitoringVersion))
		err = charts.InstallRancherMonitoringChart(clientWithSession, monitoringInstOpts, monitoringOpts)
		if err != nil {
			e2e.Failf("Failed to install the monitoring chart. Error: %v", err)
		}
	})

	It("[QASE-8321] Install an older version of the prometheus federator chart", Label("promfed", "beforeUpgrade"), func() {
		testCaseID = 8321
		e2e.Logf("Getting prometheus federator Older Version")
		promFedChartList, err := clientWithSession.Catalog.GetListChartVersions(charts.PrometheusFederatorName, catalog.RancherChartRepo)

		Expect(err).NotTo(HaveOccurred())
		e2e.Logf("Chart List: %v", promFedChartList)

		if len(promFedChartList) <= 1 {
			Skip("Not enough older versions found for the prometheus federator chart to proceed with installation")
		}
		promFedVersion := promFedChartList[1]

		prometheusFederatorChartInstallOption := &charts.InstallOptions{
			Cluster:   cluster,
			Version:   promFedVersion,
			ProjectID: project.ID,
		}

		prometheusFeatureOption := &charts.PrometheusFederatorOpts{
			EnablePodSecurity: false,
		}

		e2e.Logf("Retrieved prometheus federator chart version to install: %v", promFedVersion)

		By(fmt.Sprintf("Installing prometheus federator chart with an %v version", promFedVersion))
		err = charts.InstallPrometheusFederatorChart(clientWithSession, prometheusFederatorChartInstallOption, prometheusFeatureOption)
		if err != nil {
			e2e.Failf("Failed to install the prometheus federator chart. Error: %v", err)
		}
	})

	It("[QASE-3896] Install an older version of the Logging chart", Label("logging", "beforeUpgrade"), func() {
		testCaseID = 3896

		e2e.Logf("Getting Logging Older Version")
		LoggingVersionChartList, err := clientWithSession.Catalog.GetListChartVersions(charts.RancherLoggingName, catalog.RancherChartRepo)
		Expect(err).NotTo(HaveOccurred())
		e2e.Logf("Chart List: %v", LoggingVersionChartList)

		if len(LoggingVersionChartList) <= 1 {
			Skip("Not enough older versions found for the Logging chart to proceed with installation")
		}
		loggingVersion := LoggingVersionChartList[1]

		loggingInstOpts := &charts.InstallOptions{
			Cluster:   cluster,
			Version:   loggingVersion,
			ProjectID: project.ID,
		}

		loggingOpts := &charts.RancherLoggingOpts{
			AdditionalLoggingSources: true,
		}

		e2e.Logf("Retrieved logging chart version to install: %v", loggingVersion)

		By(fmt.Sprintf("Installing logging chart with an %v version", loggingVersion))
		err = charts.InstallRancherLoggingChart(clientWithSession, loggingInstOpts, loggingOpts)
		if err != nil {
			e2e.Failf("Failed to install the logging chart. Error: %v", err)
		}
	})

})
