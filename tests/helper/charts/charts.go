package charts

import (
	"strings"
	"time"

	"github.com/rancher/shepherd/extensions/clusters"

	v3 "github.com/rancher/rancher/pkg/apis/management.cattle.io/v3"
	"github.com/rancher/shepherd/pkg/api/steve/catalog/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// defaultRegistrySettingID is a private constant string that contains the ID of system default registry setting.
	defaultRegistrySettingID = "system-default-registry"
	// serverURLSettingID is a private constant string that contains the ID of server URL setting.
	serverURLSettingID = "server-url"
	rancherChartsName  = "rancher-charts"
	active             = "active"
)

// InstallOptions is a struct of the required options to install a chart.
type InstallOptions struct {
	Cluster   *clusters.ClusterMeta
	Version   string
	ProjectID string
}

// RancherMonitoringOpts is a struct of the required options to install Rancher Monitoring with desired chart values.
type RancherMonitoringOpts struct {
	IngressNginx      bool `json:"ingressNginx" yaml:"ingressNginx"`
	ControllerManager bool `json:"controllerManager" yaml:"controllerManager"`
	Etcd              bool `json:"etcd" yaml:"etcd"`
	Proxy             bool `json:"proxy" yaml:"proxy"`
	Scheduler         bool `json:"scheduler" yaml:"scheduler"`
}

type PrometheusFederatorOpts struct {
	EnablePodSecurity bool
}

// RancherLoggingOpts is a struct of the required options to install Rancher Logging with desired chart values.
type RancherLoggingOpts struct {
	AdditionalLoggingSources bool
}

// RancherAlertingOpts is a struct of the required options to install Rancher Alerting Drivers with desired chart values.
type RancherAlertingOpts struct {
	SMS   bool
	Teams bool
}

// GetChartCaseEndpointResult is a struct that GetChartCaseEndpoint helper function returns.
// It contains the boolean for healthy response and the request body.
type GetChartCaseEndpointResult struct {
	Ok   bool
	Body string
}

// newChartInstallAction is a private constructor that creates a payload for chart install action with given namespace, projectID, and chartInstalls.
func newChartInstallAction(namespace, projectID string, chartInstalls []types.ChartInstall) *types.ChartInstallAction {
	return &types.ChartInstallAction{
		DisableHooks:             false,
		Timeout:                  &metav1.Duration{Duration: 600 * time.Second},
		Wait:                     true,
		Namespace:                namespace,
		ProjectID:                projectID,
		DisableOpenAPIValidation: false,
		Charts:                   chartInstalls,
	}
}

// newChartInstall is a private constructor that creates a chart install with given chart values that can be used for chart install action.
func newChartInstall(name, version, clusterID, clusterName, url, repoName, projectID, defaultRegistry string, chartValues map[string]interface{}) *types.ChartInstall {
	chartInstall := types.ChartInstall{
		Annotations: map[string]string{
			"catalog.cattle.io/ui-source-repo":      repoName,
			"catalog.cattle.io/ui-source-repo-type": "cluster",
		},
		ChartName:   name,
		ReleaseName: name,
		Version:     version,
		Values: v3.MapStringInterface{
			"global": map[string]interface{}{
				"cattle": map[string]string{
					"clusterId":             clusterID,
					"clusterName":           clusterName,
					"rkePathPrefix":         "",
					"rkeWindowsPathPrefix":  "",
					"systemDefaultRegistry": defaultRegistry,
					"url":                   url,
					"systemProjectId":       strings.TrimPrefix(projectID, "local:"),
				},
				"systemDefaultRegistry": defaultRegistry,
			},
		},
	}

	// Add the prometheus-node-exporter hostRootFsMount configuration
	chartInstall.Values["prometheus-node-exporter"] = map[string]interface{}{
		"hostRootFsMount": map[string]interface{}{
			"enabled": false,
		},
	}

	for k, v := range chartValues {
		chartInstall.Values[k] = v
	}

	return &chartInstall
}
