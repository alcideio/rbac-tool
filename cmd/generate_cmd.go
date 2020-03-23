package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/alcideio/rbac-tool/kube"
)

func NewCommandGenerateClusterRole() *cobra.Command {

	clusterContext := ""
	generateKind := ""
	allowedGroups := []string{}
	//expandGroups := []string{}
	allowedVerb := []string{}
	denyResources := []string{}

	// Support overrides
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "Generate Role or ClusterRole and reduce the use of wildcards",
		Long: `
Generate Role or ClusterRole resource while reducing the use of wildcards.

rbac-tool read from the Kubernetes discovery API the available API Groups and resources, 
and based on the command line options, generate an explicit Role/ClusterRole that avoid wildcards

Examples:

# Generate a Role with read-only (get,list) excluding secrets (core group) and ingresses (extensions group) 
rbac-tool gen --generated-type=Role --deny-resources=secrets.,ingresses.extensions --allowed-verbs=get,list

# Generate a Role with read-only (get,list) excluding secrets (core group) from core group, admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io
rbac-tool gen --generated-type=ClusterRole --deny-resources=secrets., --allowed-verbs=get,list  --allowed-groups=,admissionregistration.k8s.io,storage.k8s.io,networking.k8s.io


`,
		Hidden: false,
		RunE: func(c *cobra.Command, args []string) error {

			computedPolicyRules, err := generateRules(clusterContext, sets.NewString(denyResources...), sets.NewString(allowedGroups...), sets.NewString(allowedVerb...))
			if err != nil {
				return err
			}

			var obj runtime.Object

			if generateKind == "ClusterRole" {
				obj = &rbacv1.ClusterRole{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ClusterRole",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "custom-cluster-role",
					},
					Rules: computedPolicyRules,
				}
			} else {
				obj = &rbacv1.Role{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Role",
						APIVersion: "rbac.authorization.k8s.io/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "custom-role",
						Namespace: "mynamespace",
					},
					Rules: computedPolicyRules,
				}
			}

			serializer := k8sJson.NewSerializerWithOptions(k8sJson.DefaultMetaFactory, nil, nil, k8sJson.SerializerOptions{Yaml: true, Pretty: true, Strict: true})
			var writer = bytes.NewBufferString("")
			err = serializer.Encode(obj, writer)
			if err != nil {
				return err
			}

			println(writer.String())

			return nil
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&generateKind, "generated-type", "t", "ClusterRole", "Role or ClusteRole")
	flags.StringVarP(&clusterContext, "cluster-context", "c", "", "Cluster.use 'kubectl config get-contexts' to list available contexts")
	//flags.StringSliceVarP(&expandGroups, "expand-groups", "g", []string{""},  "Comma separated list of API groups we would like to list all resource kinds rather than using wild cards '*'")
	flags.StringSliceVar(&allowedGroups, "allowed-groups", []string{"*"}, "Comma separated list of API groups we would like to allow '*'")
	flags.StringSliceVar(&allowedVerb, "allowed-verbs", []string{"*"}, "Comma separated list of verbs to include. To include all use '*'")
	flags.StringSliceVar(&denyResources, "deny-resources", []string{""}, "Comma separated list of resource.group - for example secret. to deny secret (core group) access")

	return cmd
}

func generateRules(clusterContext string, denyResources sets.String, includeGroups sets.String, allowedVerbs sets.String) ([]rbacv1.PolicyRule, error) {
	errs := []error{}
	kubeClient, err := kube.NewClient(clusterContext)
	if err != nil {
		return nil, fmt.Errorf("Failed to create kubernetes client - %v", err)
	}

	computedPolicyRules := make([]rbacv1.PolicyRule, 0)

	//processedGroups := sets.NewString()

	for _, apiGroup := range kubeClient.ServerPreferredResources {

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

		//Skip API Group versions (RBAC ignore API version)
		//if processedGroups.Has(gv.Group) {
		//	continue
		//}

		//Skip API Group entirely if *.APIGroup was specified
		if denyResources.Has(fmt.Sprintf("*.%v", strings.ToLower(gv.Group))) {
			continue
		}

		//processedGroups.Insert(gv.Group)

		resourceList := make([]string, 0)
		uniqueVerbs := sets.NewString()

		for _, kind := range apiGroup.APIResources {

			if denyResources.Has(fmt.Sprintf("%v.%v", strings.ToLower(kind.Name), strings.ToLower(gv.Group))) {
				continue
			}

			resourceList = append(resourceList, kind.Name)

			if allowedVerbs.Has(rbacv1.VerbAll) {
				uniqueVerbs.Insert(rbacv1.VerbAll)
				continue
			}

			for _, verb := range kind.Verbs {
				if allowedVerbs.Has(verb) || allowedVerbs.Has(rbacv1.VerbAll) {
					uniqueVerbs.Insert(strings.ToLower(verb))
				}
			}
		}

		var newPolicyRule *rbacv1.PolicyRule

		if len(resourceList) == 0 {
			continue
		}

		//if len(apiGroup.APIResources) == len(resourceList) {
		//	resourceList = []string{"*"}
		//}

		newPolicyRule = &rbacv1.PolicyRule{
			APIGroups: []string{gv.Group},
			Verbs:     uniqueVerbs.List(),
			Resources: resourceList,
		}

		computedPolicyRules = append(computedPolicyRules, *newPolicyRule)
	}

	return computedPolicyRules, errors.NewAggregate(errs)
}
