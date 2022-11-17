package cmd

import (
	"fmt"
	"github.com/kylelemons/godebug/pretty"
	"k8s.io/klog"
	"strings"

	"github.com/spf13/cobra"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/alcideio/rbac-tool/pkg/kube"
)

func NewCommandGenerateShowPermissions() *cobra.Command {

	clusterContext := ""
	generateKind := "ClusterRole"
	forGroups := []string{"*"}
	withVerb := []string{"*"}

	// Support overrides
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Generate ClusterRole with all available permissions from the target cluster",
		Long: `
Generate sample ClusterRole with all available permissions from the target cluster.

rbac-tool read from the Kubernetes discovery API the available API Groups and resources, 
and based on the command line options, generate an explicit ClusterRole with available resource permissions.

Examples:

# Generate a ClusterRole with all the available permissions for core and apps api groups
rbac-tool show  --for-groups=,apps


`,
		Hidden: false,
		RunE: func(c *cobra.Command, args []string) error {
			kubeClient, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("Failed to create kubernetes client - %v", err)
			}

			_, allResources, err := kubeClient.Client.Discovery().ServerGroupsAndResources()
			if err != nil {
				return fmt.Errorf("failed to read ServerGroupsAndResources - %v", err)
			}

			preferredResources, err := kubeClient.Client.Discovery().ServerPreferredResources()
			if err != nil {
				return fmt.Errorf("failed to read ServerPreferredResources - %v", err)
			}

			klog.V(7).Infof(">>>>> preferred Resources \n%v\n>>>>>", pretty.Sprint(preferredResources))

			preferredApiGroups := sets.NewString()
			for _, apiGroup := range preferredResources {
				klog.V(5).Infof("Add preferred ApiGroups: [%v]", strings.ToLower(apiGroup.GroupVersion))
				preferredApiGroups.Insert(strings.ToLower(apiGroup.GroupVersion))
			}

			klog.V(7).Infof(">>>>> All Resources \n%v\n>>>>>", pretty.Sprint(allResources))
			//if true {
			//	return nil
			//}

			computedPolicyRules, err := generateRulesWithSubResources(allResources, preferredApiGroups, sets.NewString(), sets.NewString(forGroups...), sets.NewString(withVerb...))
			if err != nil {
				return err
			}

			obj, err := generateRole(generateKind, computedPolicyRules)
			if err != nil {
				return err
			}

			println(obj)

			return nil
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&clusterContext, "cluster-context", "c", "", "Cluster.use 'kubectl config get-contexts' to list available contexts")
	flags.StringSliceVar(&forGroups, "for-groups", []string{"*"}, "Comma separated list of API groups we would like to show the permissions")
	flags.StringSliceVar(&withVerb, "with-verbs", []string{"*"}, "Comma separated list of verbs to include. To include all use '*'")

	return cmd
}

func generateRulesWithSubResources(apiresourceList []*metav1.APIResourceList, preferredApiGroups sets.String, denyResources sets.String, includeGroups sets.String, allowedVerbs sets.String) ([]rbacv1.PolicyRule, error) {
	errs := []error{}

	computedPolicyRules := make([]rbacv1.PolicyRule, 0)

	processedResources := sets.NewString()

	for _, apiGroup := range apiresourceList {

		if !preferredApiGroups.Has(strings.ToLower(apiGroup.GroupVersion)) {
			klog.V(5).Infof("Skip ApiGroups: [%v]", strings.ToLower(apiGroup.GroupVersion))
			continue
		}

		// rbac rules only look at API group names, not name + version
		gv, err := schema.ParseGroupVersion(apiGroup.GroupVersion)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		//Skip the API Groups for specific
		if !includeGroups.Has(gv.Group) && !includeGroups.Has(rbacv1.APIGroupAll) {
			continue
		}

		//Skip API Group entirely if *.APIGroup was specified
		if denyResources.Has(fmt.Sprintf("*.%v", strings.ToLower(gv.Group))) {
			continue
		}

		//processedGroups.Insert(gv.Group)

		for _, kind := range apiGroup.APIResources {

			if denyResources.Has(fmt.Sprintf("%v.%v", strings.ToLower(kind.Name), strings.ToLower(gv.Group))) {
				continue
			}

			apiResouceGVK := schema.GroupVersionResource{Group: kind.Group, Version: kind.Version, Resource: kind.Name}

			//Skip API Group versions (RBAC ignore API version)
			if processedResources.Has(apiResouceGVK.String()) {
				klog.V(5).Infof("Skp ApiGroups: [%v]", apiResouceGVK.String())
				continue
			}

			klog.V(5).Infof("Add ApiGroups: [%v]", apiResouceGVK.String())
			processedResources.Insert(apiResouceGVK.String())

			var newPolicyRule *rbacv1.PolicyRule
			var uniqueVerbs sets.String

			uniqueVerbs = sets.NewString()
			for _, verb := range kind.Verbs {
				if allowedVerbs.Has(verb) || allowedVerbs.Has(rbacv1.VerbAll) {
					uniqueVerbs.Insert(verb)
				}
			}

			if uniqueVerbs.Len() > 0 {
				newPolicyRule = &rbacv1.PolicyRule{
					APIGroups: []string{gv.Group},
					Verbs:     uniqueVerbs.List(),
					Resources: []string{kind.Name},
				}

				computedPolicyRules = append(computedPolicyRules, *newPolicyRule)
			}

		}
	}

	return computedPolicyRules, errors.NewAggregate(errs)
}
