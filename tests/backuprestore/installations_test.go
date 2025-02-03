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

package backuprestore

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/observability-e2e/tests/helper/charts"
	localConfig "github.com/rancher/observability-e2e/tests/helper/config"
	"github.com/rancher/observability-e2e/tests/helper/utils"
	rancher "github.com/rancher/shepherd/clients/rancher"
	catalog "github.com/rancher/shepherd/clients/rancher/catalog"
	extencharts "github.com/rancher/shepherd/extensions/charts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	exampleAppProjectName = "System"
)

var _ = Describe("Backup and Restore Chart Installation Test Suite", func() {
	var (
		clientWithSession   *rancher.Client
		err                 error
		backupRestoreConfig *localConfig.BackupRestoreConfig
	)
	backupRestoreConfig = &localConfig.BackupRestoreConfig{}

	JustBeforeEach(func() {
		By("Creating a client session")
		clientWithSession, err = client.WithSession(sess)
		Expect(err).NotTo(HaveOccurred())
	})

	It("Install the latest backup and restore chart", Label("LEVEL0", "backup-restore", "installation"), func() {
		By("Checking if the backup and restore chart is already installed")
		initialBackupRestoreChart, err := extencharts.GetChartStatus(clientWithSession, project.ClusterID, charts.RancherBackupRestoreNamespace, charts.RancherBackupRestoreName)
		Expect(err).NotTo(HaveOccurred())
		if initialBackupRestoreChart.IsAlreadyInstalled {
			e2e.Logf("Backup and Restore chart is already installated in project: %v", exampleAppProjectName)
		}

		if !initialBackupRestoreChart.IsAlreadyInstalled {
			By("Get the latest version of the backup and restore chart")
			latestBackupRestoreVersion, err := clientWithSession.Catalog.GetLatestChartVersion(charts.RancherBackupRestoreName, catalog.RancherChartRepo)
			Expect(err).NotTo(HaveOccurred())
			e2e.Logf("Retrieved latest backup-restore chart version to install: %v", latestBackupRestoreVersion)

			backuprestoreInstOpts := &charts.InstallOptions{
				Cluster:   cluster,
				Version:   latestBackupRestoreVersion,
				ProjectID: project.ID,
			}

			// Use the common utility function to load and unmarshal the configuration
			err = utils.LoadConfigIntoStruct(charts.BackupRestoreConfigurationFileKey, backupRestoreConfig)
			if err != nil {
				e2e.Logf("Error loading config: %v\n", err)
				return
			}

			backuprestoreOpts := &charts.RancherBackupRestoreOpts{
				VolumeName:                backupRestoreConfig.VolumeName,
				BucketName:                backupRestoreConfig.S3BucketName,
				CredentialSecretName:      charts.SecretName,
				CredentialSecretNamespace: backupRestoreConfig.CredentialSecretNamespace,
				Enabled:                   true,
				Endpoint:                  backupRestoreConfig.S3Endpoint,
				Folder:                    backupRestoreConfig.S3FolderName,
				Region:                    backupRestoreConfig.S3Region,
			}
			By("Create Opaque Secret with S3 Credentials")
			_, err = charts.CreateOpaqueS3Secret(client.Steve, backupRestoreConfig)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Installing the version %s for the backup restore", latestBackupRestoreVersion))

			err = charts.InstallRancherBackupRestoreChart(clientWithSession, backuprestoreInstOpts, backuprestoreOpts, true)
			if err != nil {
				e2e.Failf("Failed to install the backup-restore chart. Error: %v", err)
			}

			By("Waiting for backup-restore chart deployments to have expected replicas")
			errDeployChan := make(chan error, 1)
			go func() {
				err = extencharts.WatchAndWaitDeployments(clientWithSession, project.ClusterID, charts.RancherBackupRestoreNamespace, metav1.ListOptions{})
				errDeployChan <- err
			}()

			select {
			case err := <-errDeployChan:
				Expect(err).NotTo(HaveOccurred())
			case <-time.After(2 * time.Minute):
				e2e.Failf("Timeout waiting for WatchAndWaitDeployments to complete")
			}
		}

		By("Un-install the rancher backup-restore chart")
		err = charts.UninstallBackupRestoreChart(clientWithSession, project.ClusterID, charts.RancherBackupRestoreNamespace)
		if err != nil {
			e2e.Failf("Failed to un-install the backup-restore chart. Error: %v", err)
		}
	})
})
