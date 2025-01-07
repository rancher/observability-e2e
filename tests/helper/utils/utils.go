package utils

import (
	"log"
	"os"

	rancher "github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/kubectl"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

func DeployPrometheusRule(mySession *rancher.Client, yamlPath string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml"}
	prometheusRuleApply, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully: %v", prometheusRuleApply)

	return nil
}

func DeployAlertManagerConfig(mySession *rancher.Client, yamlPath string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml"}
	alertManagerConfigApply, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully: %v", alertManagerConfigApply)

	return nil
}

func DeployLoggingClusterOutputAndClusterFlow(mySession *rancher.Client, yamlPath string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml"}
	loggingResources, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully : %v", loggingResources)

	return nil
}

func DeploySyslogResources(mySession *rancher.Client, yamlPath string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml"}
	syslogResources, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully : %v", syslogResources)

	return nil
}

func DeployYamlResource(mySession *rancher.Client, yamlPath string, namespace string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "apply", "-f", "/root/.kube/my-pod.yaml", "-n", namespace}
	yamlApply, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully fetchall: %v", yamlApply)

	return nil
}

func DeleteYamlResource(mySession *rancher.Client, yamlPath string, namespace string) error {

	yamlContent, err := os.ReadFile(yamlPath)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", yamlPath, err)
	}

	importYamlInput := &management.ImportClusterYamlInput{
		YAML: string(yamlContent),
	}

	apply := []string{"kubectl", "delete", "-f", "/root/.kube/my-pod.yaml", "-n", namespace}
	yamlApply, err := kubectl.Command(mySession, importYamlInput, "local", apply, "")
	if err != nil {
		return err
	}
	e2e.Logf("Successfully fetchall: %v", yamlApply)

	return nil
}
