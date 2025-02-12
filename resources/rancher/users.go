package rancher

import (
	"context"
	"fmt"
	"strings"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/unstructured"
	"github.com/rancher/shepherd/extensions/users"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/kubectl/pkg/scheme"
)

const projectName = "testproject-"

var (
	resourceCount = 2
	rules         = []management.PolicyRule{
		{
			APIGroups: []string{"management.cattle.io"},
			Resources: []string{"projects"},
			Verbs:     []string{"backupRole"},
		},
	}
	// NamespaceGroupVersionResource is the required Group Version Resource for accessing namespaces in a cluster,
	// using the dynamic client.
	NamespaceGroupVersionResource = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}
)

// NewProjectConfig is a constructor that creates a project template
func NewProjectConfig(clusterID string) *management.Project {
	return &management.Project{
		ClusterID: clusterID,
		Name:      namegen.AppendRandomString(projectName),
	}
}

// CreateNamespace creates a namespace in a Rancher-managed Kubernetes cluster
func CreateNamespace(client *rancher.Client, clusterID, projectName, namespaceName, containerDefaultResourceLimit string, labels, annotations map[string]string) (*coreV1.Namespace, error) {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Add Rancher-specific annotations
	if containerDefaultResourceLimit != "" {
		annotations["field.cattle.io/containerDefaultResourceLimit"] = containerDefaultResourceLimit
	}
	if projectName != "" {
		annotations["field.cattle.io/projectId"] = fmt.Sprintf("%s:%s", clusterID, projectName)
	}

	// Define the namespace object
	namespace := &coreV1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespaceName,
			Annotations: annotations,
			Labels:      labels,
		},
	}

	// Get Rancher dynamic client for the target cluster
	dynamicClient, err := client.GetDownStreamClusterClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Rancher cluster client: %w", err)
	}

	// Create the namespace using the dynamic client
	namespaceResource := dynamicClient.Resource(NamespaceGroupVersionResource).Namespace("")
	unstructuredResp, err := namespaceResource.Create(context.TODO(), unstructured.MustToUnstructured(namespace), metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	// Convert unstructured response to Namespace object
	newNamespace := &coreV1.Namespace{}
	err = scheme.Scheme.Convert(unstructuredResp, newNamespace, unstructuredResp.GroupVersionKind())
	if err != nil {
		return nil, fmt.Errorf("failed to convert response to Namespace object: %w", err)
	}

	return newNamespace, nil
}

// CreateProjectAndNamespace is a helper to create a project (norman) and a namespace in the project
func CreateProjectAndNamespace(client *rancher.Client, clusterID string) (*management.Project, *coreV1.Namespace, error) {
	createdProject, err := client.Management.Project.Create(NewProjectConfig(clusterID))
	if err != nil {
		return nil, nil, err
	}

	namespaceName := namegen.AppendRandomString("testns")
	projectName := strings.Split(createdProject.ID, ":")[1]

	createdNamespace, err := CreateNamespace(client, clusterID, projectName, namespaceName, "{}", map[string]string{}, map[string]string{})
	if err != nil {
		return nil, nil, err
	}

	return createdProject, createdNamespace, nil
}

func CreateRancherResources(client *rancher.Client, clusterID string, context string) ([]*management.User, []*management.Project, []*management.RoleTemplate, error) {
	userList := []*management.User{}
	projList := []*management.Project{}
	roleList := []*management.RoleTemplate{}

	for i := 0; i < resourceCount; i++ {
		u, err := users.CreateUserWithRole(client, users.UserConfig(), "user")
		if err != nil {
			return userList, projList, roleList, err
		}
		userList = append(userList, u)

		p, _, err := CreateProjectAndNamespace(client, clusterID)
		if err != nil {
			return userList, projList, roleList, err
		}
		projList = append(projList, p)

		rt, err := client.Management.RoleTemplate.Create(
			&management.RoleTemplate{
				Context: context,
				Name:    namegen.AppendRandomString("bro-role"),
				Rules:   rules,
			})
		if err != nil {
			return userList, projList, roleList, err
		}
		roleList = append(roleList, rt)
	}

	return userList, projList, roleList, nil
}
