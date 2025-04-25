package charts

import (
	"context"
	"encoding/json"
	"errors"
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

	// Ensure the server URL has a scheme
	serverURL := serverSetting.Value
	if !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
		// Assume HTTPS if no scheme is present
		serverURL = "https://" + serverURL
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
		serverURL,
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
		serverURL,
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

	// Start watching the App resource.
	timeoutSeconds := int64(5 * 60) // 5 minutes
	watchInterface, err := catalogClient.Apps(RancherMonitoringNamespace).Watch(context.TODO(), metav1.ListOptions{
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
			return false, fmt.Errorf("failed to install rancher-monitoring chart")
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
			return fmt.Errorf("timeout: rancher-monitoring chart was not installed within 5 minutes")
		}
		return err
	}

	// The app has been successfully deployed.
	return nil
}

func UpgradeRancherMonitoringChart(client *rancher.Client, installOptions *InstallOptions, rancherMonitoringOpts *RancherMonitoringOpts) error {
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

	// Base monitoring values configuration
	monitoringValues := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"prometheusSpec": map[string]interface{}{
				"evaluationInterval": "1m",
				"retentionSize":      "50GiB",
				"scrapeInterval":     "1m",
			},
		},
	}

	// Process monitoring options with provider-specific prefixes
	optsBytes, err := json.Marshal(rancherMonitoringOpts)
	if err != nil {
		return err
	}
	optsMap := make(map[string]interface{})
	if err := json.Unmarshal(optsBytes, &optsMap); err != nil {
		return err
	}

	for key, value := range optsMap {
		if key == "ingressNginx" && installOptions.Cluster.Provider == clusters.KubernetesProviderRKE {
			monitoringValues[key] = map[string]interface{}{"enabled": value}
			continue
		}
		prefixedKey := fmt.Sprintf("%v%v%v",
			installOptions.Cluster.Provider,
			strings.ToUpper(string(key[0])),
			key[1:],
		)
		monitoringValues[prefixedKey] = map[string]interface{}{"enabled": value}
	}

	// Create chart upgrade actions
	chartUpgrade := newChartUpgrade(
		RancherMonitoringName,
		RancherMonitoringName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverURL,
		registrySetting.Value,
		monitoringValues,
	)
	chartUpgradeCRD := newChartUpgrade(
		RancherMonitoringCRDName,
		RancherMonitoringCRDName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverURL,
		registrySetting.Value,
		nil, // CRD typically doesn't need values
	)
	chartUpgrades := []types.ChartUpgrade{*chartUpgradeCRD, *chartUpgrade}
	chartUpgradeAction := newChartUpgradeAction(RancherMonitoringNamespace, chartUpgrades)

	// Execute chart upgrade
	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}
	if err := catalogClient.UpgradeChart(chartUpgradeAction, catalog.RancherChartRepo); err != nil {
		return err
	}

	// Setup watch with timeout
	timeoutSeconds := int64(5 * 60) // 5 minute timeout
	adminClient, err := rancher.NewClient(client.RancherConfig.AdminToken, client.Session)
	if err != nil {
		return err
	}
	adminCatalogClient, err := adminClient.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	// Watch for upgrade completion in two phases
	for _, phase := range []struct {
		targetState string
		phaseName   string
	}{
		{string(catalogv1.StatusPendingUpgrade), "pending upgrade"},
		{string(catalogv1.StatusDeployed), "deployed"},
	} {
		watchInterface, err := adminCatalogClient.Apps(RancherMonitoringNamespace).Watch(
			context.TODO(),
			metav1.ListOptions{
				FieldSelector:  "metadata.name=" + RancherMonitoringName,
				TimeoutSeconds: &timeoutSeconds,
			},
		)
		if err != nil {
			return err
		}

		err = wait.WatchWait(watchInterface, func(event watch.Event) (bool, error) {
			app, ok := event.Object.(*catalogv1.App)
			if !ok {
				return false, fmt.Errorf("unexpected type %T", event.Object)
			}

			switch app.Status.Summary.State {
			case phase.targetState:
				return true, nil
			case string(catalogv1.StatusFailed):
				return false, fmt.Errorf("upgrade failed at %s phase", phase.phaseName)
			default:
				return false, nil
			}
		})

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("timeout waiting for %s state (5 minutes)", phase.phaseName)
			}
			return err
		}
	}

	return nil
}
