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
	RancherLoggingNamespace = "cattle-logging-system"
	RancherLoggingName      = "rancher-logging"
	RancherLoggingCRDName   = "rancher-logging-crd"
)

// InstallRancherLoggingChart installs the rancher-logging chart with a timeout.
func InstallRancherLoggingChart(client *rancher.Client, installOptions *InstallOptions, rancherLoggingOpts *RancherLoggingOpts) error {
	// Retrieve server and registry settings.
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		return err
	}
	registrySetting, err := client.Management.Setting.ByID(defaultRegistrySettingID)
	if err != nil {
		return err
	}

	// Prepare logging values.
	loggingValues := map[string]interface{}{
		string(installOptions.Cluster.Provider): map[string]interface{}{
			"additionalLoggingSources": map[string]interface{}{
				"enabled": rancherLoggingOpts.AdditionalLoggingSources,
			},
		},
	}

	// Create chart install configurations.
	chartInstall := newChartInstall(
		RancherLoggingName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverSetting.Value,
		rancherChartsName,
		installOptions.ProjectID,
		registrySetting.Value,
		loggingValues,
	)
	chartInstallCRD := newChartInstall(
		RancherLoggingCRDName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverSetting.Value,
		rancherChartsName,
		installOptions.ProjectID,
		registrySetting.Value,
		nil,
	)

	// Combine both chart installations.
	chartInstalls := []types.ChartInstall{*chartInstallCRD, *chartInstall}
	chartInstallAction := newChartInstallAction(RancherLoggingNamespace, installOptions.ProjectID, chartInstalls)

	// Get the catalog client for the cluster.
	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	// Install the chart.
	if err = catalogClient.InstallChart(chartInstallAction, catalog.RancherChartRepo); err != nil {
		return err
	}

	// Start watching the App resource.
	timeoutSeconds := int64(5 * 60) // 5 minutes
	watchInterface, err := catalogClient.Apps(RancherLoggingNamespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector:  "metadata.name=" + RancherLoggingName,
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
			return false, fmt.Errorf("failed to install rancher-logging chart")
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
			return fmt.Errorf("timeout: rancher-logging chart was not installed within 5 minutes")
		}
		return err
	}

	// The app has been successfully deployed.
	return nil
}

func UpgradeRancherLoggingChart(client *rancher.Client, installOptions *InstallOptions, rancherLoggingOpts *RancherLoggingOpts) error {
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

	// Prepare logging values.
	loggingValues := map[string]interface{}{
		string(installOptions.Cluster.Provider): map[string]interface{}{
			"additionalLoggingSources": map[string]interface{}{
				"enabled": rancherLoggingOpts.AdditionalLoggingSources,
			},
		},
	}

	// Process logging options with provider-specific prefixes
	optsBytes, err := json.Marshal(rancherLoggingOpts)
	if err != nil {
		return err
	}
	optsMap := make(map[string]interface{})
	if err := json.Unmarshal(optsBytes, &optsMap); err != nil {
		return err
	}

	for key, value := range optsMap {
		if key == "ingressNginx" && installOptions.Cluster.Provider == clusters.KubernetesProviderRKE {
			loggingValues[key] = map[string]interface{}{"enabled": value}
			continue
		}
		prefixedKey := fmt.Sprintf("%v%v%v",
			installOptions.Cluster.Provider,
			strings.ToUpper(string(key[0])),
			key[1:],
		)
		loggingValues[prefixedKey] = map[string]interface{}{"enabled": value}
	}

	// Create chart upgrade actions
	chartUpgrade := newChartUpgrade(
		RancherLoggingName,
		RancherLoggingName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverURL,
		registrySetting.Value,
		loggingValues,
	)
	chartUpgradeCRD := newChartUpgrade(
		RancherLoggingCRDName,
		RancherLoggingCRDName,
		installOptions.Version,
		installOptions.Cluster.ID,
		installOptions.Cluster.Name,
		serverURL,
		registrySetting.Value,
		nil, // CRD typically doesn't need values
	)
	chartUpgrades := []types.ChartUpgrade{*chartUpgradeCRD, *chartUpgrade}
	chartUpgradeAction := newChartUpgradeAction(RancherLoggingNamespace, chartUpgrades)

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
		watchInterface, err := adminCatalogClient.Apps(RancherLoggingNamespace).Watch(
			context.TODO(),
			metav1.ListOptions{
				FieldSelector:  "metadata.name=" + RancherLoggingName,
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
