package charts

import (
	"context"
	"fmt"

	localConfig "github.com/rancher/observability-e2e/tests/helper/config"
	catalogv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/rancher/tests/v2/actions/secrets"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	v1 "github.com/rancher/shepherd/clients/rancher/v1"
	"github.com/rancher/shepherd/pkg/api/steve/catalog/types"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	RancherBackupRestoreNamespace     = "cattle-resources-system"
	RancherBackupRestoreName          = "rancher-backup"
	RancherBackupRestoreCRDName       = "rancher-backup-crd"
	BackupRestoreConfigurationFileKey = "../helper/yamls/inputBackupRestoreConfig.yaml"
)

var (
	SecretName = namegen.AppendRandomString("bro-secret")
)

// InstallRancherBackupRestoreChart installs the Rancher backup/restore chart with optional storage configuration.
func InstallRancherBackupRestoreChart(client *rancher.Client, installOptions *InstallOptions, rancherBackupRestoreOpts *RancherBackupRestoreOpts, withStorage bool) error {
	// Retrieve the server URL setting from Rancher.
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		return err
	}

	// Prepare the payload for chart installation.
	backupChartInstallActionPayload := &PayloadOpts{
		InstallOptions: *installOptions,
		Name:           RancherBackupRestoreName,
		Namespace:      RancherBackupRestoreNamespace,
		Host:           serverSetting.Value,
	}
	chartInstallAction := newBackupChartInstallAction(backupChartInstallActionPayload, withStorage, rancherBackupRestoreOpts)

	// Get the catalog client for the specified cluster.
	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	// Install the chart using the catalog client.
	if err = catalogClient.InstallChart(chartInstallAction, catalog.RancherChartRepo); err != nil {
		return err
	}

	// Watch for the App resource to ensure successful deployment.
	watchInterface, err := catalogClient.Apps(RancherBackupRestoreNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  metadataName + RancherBackupRestoreName,
		TimeoutSeconds: &FiveMinuteTimeout,
	})
	if err != nil {
		return err
	}

	// Check function to validate the state of the app during the watch.
	checkFunc := func(event watch.Event) (bool, error) {
		app, ok := event.Object.(*catalogv1.App)
		if !ok {
			return false, fmt.Errorf("unexpected type %T", event.Object)
		}

		// Check the deployment state of the app.
		state := app.Status.Summary.State
		switch state {
		case string(catalogv1.StatusDeployed):
			return true, nil // Deployment succeeded.
		case string(catalogv1.StatusFailed):
			return false, fmt.Errorf("failed to install rancher-backup-restore chart") // Deployment failed.
		default:
			return false, nil // Continue waiting.
		}
	}

	// Wait for the app to be successfully deployed.
	err = wait.WatchWait(watchInterface, checkFunc)
	if err != nil {
		if err.Error() == wait.TimeoutError {
			return fmt.Errorf("timeout: rancher-backup-restore chart was not installed within 5 minutes")
		}
		return err
	}

	return nil // Successful installation.
}

// CreateOpaqueS3Secret creates an opaque Kubernetes secret for S3 credentials.
func CreateOpaqueS3Secret(steveClient *v1.Client, backupRestoreConfig *localConfig.BackupRestoreConfig) (string, error) {
	// Define the secret template with S3 access and secret keys.
	secretTemplate := secrets.NewSecretTemplate(
		SecretName,
		backupRestoreConfig.CredentialSecretNamespace,
		map[string][]byte{
			"accessKey": []byte(backupRestoreConfig.AccessKey),
			"secretKey": []byte(backupRestoreConfig.SecretKey),
		},
		corev1.SecretTypeOpaque,
	)
	// Create the secret using the Steve client.
	createdSecret, err := steveClient.SteveType(secrets.SecretSteveType).Create(secretTemplate)

	return createdSecret.Name, err
}

// newBackupChartInstallAction prepares the chart installation action with storage and payload options.
func newBackupChartInstallAction(p *PayloadOpts, withStorage bool, rancherBackupRestoreOpts *RancherBackupRestoreOpts) *types.ChartInstallAction {
	// Configure backup values if storage is enabled.
	backupValues := map[string]interface{}{}
	if withStorage {
		backupValues = map[string]any{
			"s3": map[string]any{
				"bucketName":                rancherBackupRestoreOpts.BucketName,
				"credentialSecretName":      rancherBackupRestoreOpts.CredentialSecretName,
				"credentialSecretNamespace": rancherBackupRestoreOpts.CredentialSecretNamespace,
				"enabled":                   rancherBackupRestoreOpts.Enabled,
				"endpoint":                  rancherBackupRestoreOpts.Endpoint,
				"folder":                    rancherBackupRestoreOpts.Folder,
				"region":                    rancherBackupRestoreOpts.Region,
			},
		}
	}

	// Prepare the chart installation actions for the backup and its CRDs.
	chartInstall := newChartInstall(
		p.Name,
		p.InstallOptions.Version,
		p.InstallOptions.Cluster.ID,
		p.InstallOptions.Cluster.Name,
		p.Host,
		rancherChartsName,
		p.ProjectID,
		p.DefaultRegistry,
		backupValues,
	)

	chartInstallCRD := newChartInstall(
		p.Name+"-crd",
		p.InstallOptions.Version,
		p.InstallOptions.Cluster.ID,
		p.InstallOptions.Cluster.Name,
		p.Host,
		rancherChartsName,
		p.ProjectID,
		p.DefaultRegistry,
		nil,
	)

	chartInstalls := []types.ChartInstall{*chartInstallCRD, *chartInstall}

	// Combine the chart installs into a single installation action.
	chartInstallAction := newChartInstallAction(p.Namespace, p.ProjectID, chartInstalls)

	return chartInstallAction
}

// Function to uninstall the backup-restore charts
func UninstallBackupRestoreChart(client *rancher.Client, clusterID string, namespace string) error {
	chartNames := []string{RancherBackupRestoreName, RancherBackupRestoreCRDName}

	for _, chartName := range chartNames {
		err := UninstallChart(client, clusterID, chartName, namespace)
		if err != nil {
			e2e.Failf("Failed to uninstall the chart %s. Error: %v", chartName, err)
			return err // Stop on first failure
		}
	}
	return nil
}
