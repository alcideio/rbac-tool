package kube

import (
	"github.com/rs/zerolog/log"
	"strings"

	v1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/version"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure" // auth for AKS clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/exec"  // auth for OIDC
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"   // auth for GKE clusters
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"  // auth for OIDC
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubeClient struct {
	Client *clientset.Clientset

	Config *restclient.Config

	// ServerPreferredResources returns the supported resources with the version preferred by the
	// server.
	ServerPreferredResources []*metav1.APIResourceList
	masterVersion            *version.Info
}

func NewClient(context string) (*KubeClient, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	// if you want to change the loading rules (which files in which order), you can do so here

	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: context,
	}
	// if you want to change override values or bind them to flags, there are methods to help you

	var config *restclient.Config
	var err error

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	config, err = kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	preferedResource, err := client.Discovery().ServerPreferredResources()
	if err != nil {
		log.Debug().Msgf("ServerPreferredResources completed with errors %v (%v)", err, len(preferedResource))
	}

	if preferedResource == nil {
		preferedResource = []*metav1.APIResourceList{}
	}

	//log.Debug().Msgf("%v\n", pretty.Sprint(preferedResource))

	k8sVer, err := client.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}

	return &KubeClient{
		Client:                   client,
		ServerPreferredResources: preferedResource,
		Config:                   config,
		masterVersion:            k8sVer,
	}, nil
}

func (kubeClient *KubeClient) GetWorldPermissions() ([]rbacv1.PolicyRule, error) {
	errs := []error{}
	computedPolicyRules := make([]rbacv1.PolicyRule, 0)

	for _, apiResourceList := range kubeClient.ServerPreferredResources {
		// rbac rules only look at API group names, not name & version
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		resourceList := make([]string, 0)
		uniqueVerbs := sets.NewString()

		for _, apiResource := range apiResourceList.APIResources {
			resourceList = append(resourceList, apiResource.Name)
			for _, verb := range apiResource.Verbs {
				uniqueVerbs.Insert(strings.ToLower(verb))
			}
		}

		newPolicyRule := &rbacv1.PolicyRule{
			APIGroups: []string{gv.Group},
			Verbs:     uniqueVerbs.List(),
			Resources: resourceList,
		}

		computedPolicyRules = append(computedPolicyRules, *newPolicyRule)
	}

	return computedPolicyRules, errors.NewAggregate(errs)
}

func (kubeClient *KubeClient) GetResourcesAndVerbsForGroup(apiGroup string) (sets.String, sets.String, error) {
	errs := []error{}

	resources := sets.NewString()
	verbs := sets.NewString()

	for _, apiResourceList := range kubeClient.ServerPreferredResources {
		// rbac rules only look at API group names, not name & version
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if strings.ToLower(gv.Group) != strings.ToLower(apiGroup) {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			resources.Insert(apiResource.Name)
			for _, verb := range apiResource.Verbs {
				verbs.Insert(strings.ToLower(verb))
			}
		}
	}

	return resources, verbs, errors.NewAggregate(errs)
}

func (kubeClient *KubeClient) GetVerbsForResource(apiGroup string, resource string) (sets.String, error) {
	errs := []error{}

	verbs := sets.NewString()

	for _, apiResourceList := range kubeClient.ServerPreferredResources {
		// rbac rules only look at API group names, not name & version
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if strings.ToLower(gv.Group) != strings.ToLower(apiGroup) {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			if apiResource.Name != resource {
				continue
			}

			for _, verb := range apiResource.Verbs {
				verbs.Insert(strings.ToLower(verb))
			}
		}
	}

	return verbs, errors.NewAggregate(errs)
}

func (kubeClient *KubeClient) ListPods(namespace string) ([]v1.Pod, error) {
	objs, err := kubeClient.Client.CoreV1().Pods(namespace).List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListServiceAccounts(namespace string) ([]v1.ServiceAccount, error) {
	objs, err := kubeClient.Client.CoreV1().ServiceAccounts(namespace).List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListRoles(namespace string) ([]rbacv1.Role, error) {
	objs, err := kubeClient.Client.RbacV1().Roles(namespace).List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListRoleBindings(namespace string) ([]rbacv1.RoleBinding, error) {
	objs, err := kubeClient.Client.RbacV1().RoleBindings(namespace).List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListClusterRoles() ([]rbacv1.ClusterRole, error) {
	objs, err := kubeClient.Client.RbacV1().ClusterRoles().List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListClusterRoleBindings() ([]rbacv1.ClusterRoleBinding, error) {
	objs, err := kubeClient.Client.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListPodSecurityPolicies() ([]policy.PodSecurityPolicy, error) {
	objs, err := kubeClient.Client.PolicyV1beta1().PodSecurityPolicies().List(metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}
