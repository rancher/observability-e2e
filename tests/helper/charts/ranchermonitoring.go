package charts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	catalogv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/api/steve/catalog/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	RancherMonitoringNamespace = "cattle-monitoring-system"
	RancherMonitoringName      = "rancher-monitoring"
	RancherMonitoringCRDName   = "rancher-monitoring-crd"
)

// InstallRancherMonitoringChart installs the rancher-monitoring chart with a timeout.
func InstallRancherMonitoringChart(client *rancher.Client, installOptions *InstallOptions, rancherMonitoringOpts *RancherMonitoringOpts) error {
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

	// Prepare the monitoring values with default Prometheus configurations.
	monitoringValues := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"prometheusSpec": map[string]interface{}{
				"evaluationInterval": "1m",
				"retentionSize":      "50GiB",
				"scrapeInterval":     "1m",
			},
		},
	}

	// Convert rancherMonitoringOpts to a map for easier manipulation.
	optsBytes, err := json.Marshal(rancherMonitoringOpts)
	if err != nil {
		return err
	}
	optsMap := map[string]interface{}{}
	if err = json.Unmarshal(optsBytes, &optsMap); err != nil {
		return err
	}

	// Add provider-specific options to the monitoring values.
	for key, value := range optsMap {
		var newKey string
		// Special case for "ingressNginx" when using RKE provider.
		if key == "ingressNginx" && installOptions.Cluster.Provider == clusters.KubernetesProviderRKE {
			newKey = key
		} else {
			// Format the key based on the cluster provider and option name.
			newKey = fmt.Sprintf("%v%v%v", installOptions.Cluster.Provider, strings.ToUpper(string(key[0])), key[1:])
		}
		monitoringValues[newKey] = map[string]interface{}{"enabled": value}
	}

	// Create chart install configurations for the CRD and the main chart.
	chartInstallCRD := newChartInstall(
		RancherMonitoringCRDName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverSetting.Value,
		rancherChartsName,
		installOptions.ProjectID,
		registrySetting.Value,
		nil,
	)
	chartInstall := newChartInstall(
		RancherMonitoringName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverSetting.Value,
		rancherChartsName,
		installOptions.ProjectID,
		registrySetting.Value,
		monitoringValues,
	)

	// Combine both chart installations.
	chartInstalls := []types.ChartInstall{*chartInstallCRD, *chartInstall}
	chartInstallAction := newChartInstallAction(RancherMonitoringNamespace, installOptions.ProjectID, chartInstalls)

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

	// Start polling to check the deployment status.
	err = wait.Poll(interval, timeout, func() (bool, error) {
		// Attempt to get the app from the catalog.
		app, err := catalogClient.Apps(RancherMonitoringNamespace).Get(context.TODO(), RancherMonitoringName, metav1.GetOptions{})
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
			return false, fmt.Errorf("failed to install rancher-monitoring chart")
		default:
			// The app is still deploying; continue waiting.
			return false, nil
		}
	})

	// Handle the result of the polling.
	if err != nil {
		if wait.Interrupted(err) {
			return fmt.Errorf("timeout: rancher-monitoring chart was not installed within 10 minutes")
		}
		return err
	}

	// The app has been successfully deployed.
	return nil
}
