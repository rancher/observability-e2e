package charts

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	catalogv1 "github.com/rancher/rancher/pkg/apis/catalog.cattle.io/v1"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/clients/rancher/catalog"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/api/steve/catalog/types"
	"github.com/rancher/shepherd/pkg/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	PrometheusFederatorNamespace = "cattle-monitoring-system"
	PrometheusFederatorName      = "prometheus-federator"
)

// InstallRancherMonitoringChart installs the rancher-monitoring chart with a timeout.
func InstallPrometheusFederatorChart(client *rancher.Client, installOptions *InstallOptions, prometheusFederatorOpts *PrometheusFederatorOpts) error {
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
	prometheusValues := map[string]interface{}{
		"podSecurity": map[string]interface{}{
			"enabled": prometheusFederatorOpts.EnablePodSecurity,
		},
	}

	// Create chart install configurations for the CRD and the main chart.
	chartInstall := newChartInstall(
		PrometheusFederatorName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverSetting.Value,
		rancherChartsName,
		installOptions.ProjectID,
		registrySetting.Value,
		prometheusValues,
	)

	// Combine both chart installations.
	chartInstalls := []types.ChartInstall{*chartInstall}
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

	// Start watching the App resource.
	timeoutSeconds := int64(1 * 60) // 5 minutes
	watchInterface, err := catalogClient.Apps(PrometheusFederatorNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + RancherMonitoringName,
		TimeoutSeconds: &timeoutSeconds,
	})
	if err != nil {
		return err
	}

	// Define the check function for WatchWait.
	checkFunc := func(event watch.Event) (bool, error) {
		app, ok := event.Object.(*catalogv1.App)
		if !ok {
			return false, fmt.Errorf("unexpected type %T", event.Object)
		}

		// Check the deployment status of the app.
		state := app.Status.Summary.State

		switch state {
		case string(catalogv1.StatusDeployed):
			// The app has been successfully deployed.
			return true, nil
		case string(catalogv1.StatusFailed):
			// The app failed to deploy.
			return false, fmt.Errorf("failed to install prometheus-federator chart")
		default:
			// The app is still deploying; continue waiting.
			return false, nil
		}
	}

	// Use WatchWait to wait until the app is deployed.
	err = wait.WatchWait(watchInterface, checkFunc)

	// Handle the result.
	if err != nil {
		if err.Error() == wait.TimeoutError {
			return fmt.Errorf("timeout: prometheus-federator chart was not installed within 5 minutes")
		}
		return err
	}

	// The app has been successfully deployed.
	return nil
}

func UpgradePrometheusFederatorChart(client *rancher.Client, installOptions *InstallOptions, prometheusFederatorOpts *PrometheusFederatorOpts) error {
	// Retrieve server settings with scheme validation
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		return err
	}
	serverURL := serverSetting.Value
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		serverURL = "https://" + serverURL
	}

	// Retrieve registry settings
	registrySetting, err := client.Management.Setting.ByID(defaultRegistrySettingID)
	if err != nil {
		return err
	}

	// Prepare the values with default Prometheus configurations.
	prometheusValues := map[string]interface{}{
		"podSecurity": map[string]interface{}{
			"enabled": prometheusFederatorOpts.EnablePodSecurity,
		},
	}

	// Process monitoring options with provider-specific prefixes
	optsBytes, err := json.Marshal(prometheusFederatorOpts)
	if err != nil {
		return err
	}
	optsMap := make(map[string]interface{})
	if err := json.Unmarshal(optsBytes, &optsMap); err != nil {
		return err
	}

	for key, value := range optsMap {
		if key == "ingressNginx" && installOptions.Cluster.Provider == clusters.KubernetesProviderRKE {
			prometheusValues[key] = map[string]interface{}{"enabled": value}
			continue
		}
		prefixedKey := fmt.Sprintf("%v%v%v",
			installOptions.Cluster.Provider,
			strings.ToUpper(string(key[0])),
			key[1:],
		)
		prometheusValues[prefixedKey] = map[string]interface{}{"enabled": value}
	}

	// Create chart upgrade actions
	chartUpgrade := newChartUpgrade(
		PrometheusFederatorName,
		PrometheusFederatorName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverURL,
		registrySetting.Value,
		prometheusValues,
	)

	chartUpgrades := []types.ChartUpgrade{*chartUpgrade}
	chartUpgradeAction := newChartUpgradeAction(PrometheusFederatorNamespace, chartUpgrades)

	// Execute chart upgrade
	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}
	if err := catalogClient.UpgradeChart(chartUpgradeAction, catalog.RancherChartRepo); err != nil {
		return err
	}

	// Start watching the App resource.
	timeoutSeconds := int64(1 * 60) // 5 minutes
	watchInterface, err := catalogClient.Apps(PrometheusFederatorNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + RancherMonitoringName,
		TimeoutSeconds: &timeoutSeconds,
	})
	if err != nil {
		return err
	}

	// Define the check function for WatchWait.
	checkFunc := func(event watch.Event) (bool, error) {
		app, ok := event.Object.(*catalogv1.App)
		if !ok {
			return false, fmt.Errorf("unexpected type %T", event.Object)
		}

		// Check the deployment status of the app.
		state := app.Status.Summary.State

		switch state {
		case string(catalogv1.StatusDeployed):
			// The app has been successfully deployed.
			return true, nil
		case string(catalogv1.StatusFailed):
			// The app failed to deploy.
			return false, fmt.Errorf("failed to install prometheus-federator chart")
		default:
			// The app is still deploying; continue waiting.
			return false, nil
		}
	}

	// Use WatchWait to wait until the app is deployed.
	err = wait.WatchWait(watchInterface, checkFunc)

	// Handle the result.
	if err != nil {
		if err.Error() == wait.TimeoutError {
			return fmt.Errorf("timeout: prometheus-federator chart was not installed within 5 minutes")
		}
		return err
	}

	// The app has been successfully deployed.
	return nil
}
