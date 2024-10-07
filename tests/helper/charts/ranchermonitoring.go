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
)

const (
	RancherMonitoringNamespace = "cattle-monitoring-system"
	RancherMonitoringName      = "rancher-monitoring"
	RancherMonitoringCRDName   = "rancher-monitoring-crd"
)

// InstallRancherMonitoringChart installs the rancher-monitoring chart with a timeout.
func InstallRancherMonitoringChart(client *rancher.Client, installOptions *InstallOptions, rancherMonitoringOpts *RancherMonitoringOpts) error {
	serverSetting, err := client.Management.Setting.ByID(serverURLSettingID)
	if err != nil {
		return err
	}
	registrySetting, err := client.Management.Setting.ByID(defaultRegistrySettingID)
	if err != nil {
		return err
	}

	// Prepare monitoring values
	monitoringValues := map[string]interface{}{
		"prometheus": map[string]interface{}{
			"prometheusSpec": map[string]interface{}{
				"evaluationInterval": "1m",
				"retentionSize":      "50GiB",
				"scrapeInterval":     "1m",
			},
		},
	}
	// Add provider-specific options
	optsBytes, err := json.Marshal(rancherMonitoringOpts)
	if err != nil {
		return err
	}
	optsMap := map[string]interface{}{}
	if err = json.Unmarshal(optsBytes, &optsMap); err != nil {
		return err
	}
	for k, v := range optsMap {
		var newKey string
		if k == "ingressNginx" && installOptions.Cluster.Provider == clusters.KubernetesProviderRKE {
			newKey = k
		} else {
			newKey = fmt.Sprintf("%v%v%v", installOptions.Cluster.Provider, strings.ToUpper(string(k[0])), k[1:])
		}
		monitoringValues[newKey] = map[string]interface{}{"enabled": v}
	}

	// Create chart install action
	chartInstall := newChartInstall(RancherMonitoringName, installOptions.Version, installOptions.Cluster.ID,
		installOptions.Cluster.Name, serverSetting.Value, rancherChartsName, installOptions.ProjectID,
		registrySetting.Value, monitoringValues)
	chartInstallCRD := newChartInstall(RancherMonitoringCRDName, installOptions.Version, installOptions.Cluster.ID,
		installOptions.Cluster.Name, serverSetting.Value, rancherChartsName, installOptions.ProjectID,
		registrySetting.Value, nil)
	chartInstalls := []types.ChartInstall{*chartInstallCRD, *chartInstall}
	chartInstallAction := newChartInstallAction(RancherMonitoringNamespace, installOptions.ProjectID, chartInstalls)

	catalogClient, err := client.GetClusterCatalogClient(installOptions.Cluster.ID)
	if err != nil {
		return err
	}

	// Install the chart
	if err = catalogClient.InstallChart(chartInstallAction, catalog.RancherChartRepo); err != nil {
		return err
	}

	// Wait for the chart to be fully deployed with a timeout
	//TODO: Need to add wait poll instread of ticker
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout: rancher-monitoring chart was not installed within 10 minutes")
		case <-ticker.C:
			app, err := catalogClient.Apps(RancherMonitoringNamespace).Get(context.TODO(), RancherMonitoringName, metav1.GetOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					// App not yet created, continue waiting
					continue
				}
				return err
			}
			state := app.Status.Summary.State
			if state == string(catalogv1.StatusDeployed) {
				return nil
			}
			if state == string(catalogv1.StatusFailed) {
				return fmt.Errorf("failed to install rancher-monitoring chart")
			}
		}
	}
}
