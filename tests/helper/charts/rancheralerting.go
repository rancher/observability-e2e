package charts

import (
	"context"
	"fmt"
	"time"

	catalogv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	"github.com/rancher/shepherd/pkg/api/steve/catalog/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Updated namespace for the alerting chart
	RancherAlertingNamespace = "cattle-alerting"
	// Name of the rancher-alerting-drivers chart
	RancherAlertingName = "rancher-alerting-drivers"
)

// InstallRancherAlertingChart installs the rancher-alerting-drivers chart with a timeout.
func InstallRancherAlertingChart(client *rancher.Client, installOptions *InstallOptions, rancherAlertingOpts *RancherAlertingOpts) error {
	// Retrieve the server URL setting.
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		return err
	}

	// Retrieve the default registry setting.
	registrySetting, err := client.Management.Setting.ByID(defaultRegistrySettingID)
	if err != nil {
		return err
	}

	// Prepare the alerting values.
	alertingValues := map[string]interface{}{
		"prom2teams": map[string]interface{}{
			"enabled": rancherAlertingOpts.Teams,
		},
		"sachet": map[string]interface{}{
			"enabled": rancherAlertingOpts.SMS,
		},
	}

	// Create chart install configuration for the main chart.
	chartInstall := newChartInstall(
		RancherAlertingName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverSetting.Value,
		rancherChartsName,
		installOptions.ProjectID,
		registrySetting.Value,
		alertingValues,
	)

	// Combine chart installations.
	chartInstalls := []types.ChartInstall{*chartInstall}
	chartInstallAction := newChartInstallAction(RancherAlertingNamespace, installOptions.ProjectID, chartInstalls)

	// Get the catalog client for the cluster.
	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	// Install the chart using the catalog client.
	if err = catalogClient.InstallChart(chartInstallAction, catalog.RancherChartRepo); err != nil {
		return err
	}

	// Define the polling interval and timeout duration.
	interval := 10 * time.Second
	timeout := 10 * time.Minute

	// Start polling to check the deployment status using wait.Poll.
	err = wait.Poll(interval, timeout, func() (bool, error) {
		// Attempt to get the app from the catalog.
		app, err := catalogClient.Apps(RancherAlertingNamespace).Get(context.TODO(), RancherAlertingName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				// The app is not yet created; continue waiting.
				return false, nil
			}
			// An error occurred; stop waiting and return the error.
			return false, err
		}

		// Check the deployment status of the app.
		state := app.Status.Summary.State

		switch state {
		case string(catalogv1.StatusDeployed):
			// The app has been successfully deployed.
			return true, nil
		case string(catalogv1.StatusFailed):
			// The app failed to deploy.
			return false, fmt.Errorf("failed to install rancher-alerting-drivers chart")
		default:
			// The app is still deploying; continue waiting.
			return false, nil
		}
	})

	// Handle the result of the polling.
	if err != nil {
		if err == wait.ErrWaitTimeout {
			return fmt.Errorf("timeout: rancher-alerting-drivers chart was not installed within 10 minutes")
		}
		return err
	}

	// The app has been successfully deployed.
	return nil
}
