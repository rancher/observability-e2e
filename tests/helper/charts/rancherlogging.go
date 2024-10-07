package charts

import (
	"context"
	"fmt"
	"time"

	catalogv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	"github.com/rancher/shepherd/pkg/api/steve/catalog/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RancherLoggingNamespace = "cattle-logging-system"
	RancherLoggingName      = "rancher-logging"
	RancherLoggingCRDName   = "rancher-logging-crd"
)

// InstallRancherLoggingChart installs the rancher-logging chart with a timeout.
func InstallRancherLoggingChart(client *rancher.Client, installOptions *InstallOptions, rancherLoggingOpts *RancherLoggingOpts) error {
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		return err
	}
	registrySetting, err := client.Management.Setting.ByID(defaultRegistrySettingID)
	if err != nil {
		return err
	}

	// Prepare logging values
	loggingValues := map[string]interface{}{
		string(installOptions.Cluster.Provider): map[string]interface{}{
			"additionalLoggingSources": map[string]interface{}{
				"enabled": rancherLoggingOpts.AdditionalLoggingSources,
			},
		},
	}

	// Create chart install action
	chartInstall := newChartInstall(RancherLoggingName, installOptions.Version, installOptions.Cluster.ID,
		installOptions.Cluster.Name, serverSetting.Value, rancherChartsName, installOptions.ProjectID,
		registrySetting.Value, loggingValues)
	chartInstallCRD := newChartInstall(RancherLoggingCRDName, installOptions.Version, installOptions.Cluster.ID,
		installOptions.Cluster.Name, serverSetting.Value, rancherChartsName, installOptions.ProjectID,
		registrySetting.Value, nil)
	chartInstalls := []types.ChartInstall{*chartInstallCRD, *chartInstall}
	chartInstallAction := newChartInstallAction(RancherLoggingNamespace, installOptions.ProjectID, chartInstalls)

	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	// Install the chart
	if err = catalogClient.InstallChart(chartInstallAction, catalog.RancherChartRepo); err != nil {
		return err
	}

	// Wait for the chart to be fully deployed with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout: rancher-logging chart was not installed within 10 minutes")
		case <-ticker.C:
			app, err := catalogClient.Apps(RancherLoggingNamespace).Get(context.TODO(), RancherLoggingName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			state := app.Status.Summary.State
			if state == string(catalogv1.StatusDeployed) {
				return nil
			}
			if state == string(catalogv1.StatusFailed) {
				return fmt.Errorf("failed to install rancher-logging chart")
			}
		}
	}
}
