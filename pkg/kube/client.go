package kube

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	authn "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sserrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/version"
	clientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
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
		klog.V(3).Infof("ServerPreferredResources completed with errors %v (%v)", err, len(preferedResource))
	}

	if preferedResource == nil {
		preferedResource = []*metav1.APIResourceList{}
	}

	//klog.V(8).Infof("%v\n", pretty.Sprint(preferedResource))

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
	objs, err := kubeClient.Client.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListServiceAccounts(namespace string) ([]v1.ServiceAccount, error) {
	objs, err := kubeClient.Client.CoreV1().ServiceAccounts(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListRoles(namespace string) ([]rbacv1.Role, error) {
	objs, err := kubeClient.Client.RbacV1().Roles(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListRoleBindings(namespace string) ([]rbacv1.RoleBinding, error) {
	objs, err := kubeClient.Client.RbacV1().RoleBindings(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListClusterRoles() ([]rbacv1.ClusterRole, error) {
	objs, err := kubeClient.Client.RbacV1().ClusterRoles().List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) ListClusterRoleBindings() ([]rbacv1.ClusterRoleBinding, error) {
	objs, err := kubeClient.Client.RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return objs.Items, nil
}

func (kubeClient *KubeClient) TokenReview(token string) (authn.UserInfo, error) {
	tokenReview, err := kubeClient.Client.AuthenticationV1().TokenReviews().Create(
		context.Background(),
		&authn.TokenReview{
			Spec: authn.TokenReviewSpec{Token: token},
		},
		metav1.CreateOptions{},
	)

	if err != nil {
		if k8sserrs.IsForbidden(err) {
			//definitely bad ... but at least give some sense to the user
			usernameFromErrorRE := regexp.MustCompile(`^.* User "(.*)" cannot .*$`)
			username := usernameFromErrorRE.ReplaceAllString(err.Error(), "$1")
			return authn.UserInfo{Username: username}, nil
		}
		return authn.UserInfo{}, err
	}

	if tokenReview.Status.Error != "" {
		return authn.UserInfo{}, fmt.Errorf(tokenReview.Status.Error)
	}

	return tokenReview.Status.User, nil
}

func (kubeClient *KubeClient) Resolve(verb, groupresource string, subResource string) (schema.GroupResource, error) {
	gr := schema.ParseGroupResource(groupresource)

	klog.V(8).Infof("resolving %v", gr.String())

	for _, apiResourceList := range kubeClient.ServerPreferredResources {
		// rbac rules only look at API group names, not name & version
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			klog.V(8).Infof("failed to parse %v - %v", apiResourceList.GroupVersion, err)
			continue
		}

		if gr.Group != "" && strings.ToLower(gv.Group) != strings.ToLower(gr.Group) {
			klog.V(8).Infof("skip - gr=%v,gv=%v", gr.String(), gr.String())
			continue
		}

		//We are looking at the correct API Group
		//Look at the resource kinds

		for _, apiResource := range apiResourceList.APIResources {

			possibleNames := sets.NewString(apiResource.ShortNames...)
			possibleNames.Insert(strings.ToLower(apiResource.Name))
			possibleNames.Insert(strings.ToLower(apiResource.Kind))

			if !possibleNames.Has(strings.ToLower(gr.Resource)) {
				klog.V(8).Infof("skip - gr=%v NOT in [%v]", gr.String(), strings.Join(possibleNames.List(), ","))
				continue
			}

			r := schema.GroupResource{
				Group:    strings.ToLower(gv.Group),
				Resource: strings.ToLower(apiResource.Name),
			}

			//Special Verbs
			switch verb {
			case "bind", "escalate":
				//bind: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#restrictions-on-role-binding-creation-or-update
				//escalate: https://kubernetes.io/docs/reference/access-authn-authz/rbac/#restrictions-on-role-creation-or-update
				if gv.Group == "rbac.authorization.k8s.io" &&
					(possibleNames.Has("clusterroles") || possibleNames.Has("roles")) {
					//We have a match
					return r, nil
				}
			case "impersonate":
				if gv.Group == "" &&
					(possibleNames.Has("users") || possibleNames.Has("groups") || possibleNames.Has("serviceaccounts")) {
					//We have a match
					return r, nil
				}
			}

			possibleVerbs := sets.NewString(apiResource.Verbs...)
			if !possibleVerbs.Has(strings.ToLower(verb)) {
				klog.V(8).Infof("skip - gr=%v '%v' is not in [%v]", gr.String(), verb, strings.Join(possibleVerbs.List(), ","))
				return r, fmt.Errorf("The verb '%s' is not supported by %v", strings.ToLower(verb), r.String())
			}

			//We have a match
			return r, nil
		}
	}

	return schema.GroupResource{}, fmt.Errorf("Failed find a matching API resource")
}
