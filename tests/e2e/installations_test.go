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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/rancher/tests/v2/actions/charts"
	"github.com/rancher/rancher/tests/v2/actions/registries"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	extencharts "github.com/rancher/shepherd/extensions/charts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Observability Installation Test Suite", func() {

	It("Should install monitoring chart if not already installed", func() {
		clientWithSession, err := client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())

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
})

func TestGinkgoSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Installation Test Suite")
}
