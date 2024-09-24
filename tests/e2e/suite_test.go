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
	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/session"
)

const exampleAppProjectName = "demo-project"

var (
	client          *rancher.Client
	sess            *session.Session
	project         *management.Project
	cluster         *clusters.ClusterMeta
	registrySetting *management.Setting
)

func FailWithReport(message string, callerSkip ...int) {
	// Ensures the correct line numbers are reported
	Fail(message, callerSkip[0]+1)
}

// func TestE2E(t *testing.T) {
// 	RegisterFailHandler(FailWithReport)
// 	RunSpecs(t, "Monitoring End-To-End Test Suite")
// }

// This setup will run once for the entire test suite
var _ = BeforeSuite(func() {
	var err error
	testSession := session.NewSession()
	sess = testSession

	client, err = rancher.NewClient("", testSession)
	Expect(err).NotTo(HaveOccurred())

	// Get clusterName from config yaml
	clusterName := client.RancherConfig.ClusterName
	Expect(clusterName).NotTo(BeEmpty(), "Cluster name to install is not set")

	// Get cluster meta
	cluster, err = clusters.NewClusterMeta(client, clusterName)
	Expect(err).NotTo(HaveOccurred())

	// Get Server and Registry Setting Values
	registrySetting, err = client.Management.Setting.ByID("system-default-registry")
	Expect(err).NotTo(HaveOccurred())

	// Create project
	projectConfig := &management.Project{
		ClusterID: cluster.ID,
		Name:      exampleAppProjectName,
	}
	project, err = client.Management.Project.Create(projectConfig)
	Expect(err).NotTo(HaveOccurred())
	Expect(project.Name).To(Equal(exampleAppProjectName))
})

// This teardown will run once after all the tests in the suite are done
// var _ = AfterSuite(func() {
// 	sess.Cleanup()
// })
