package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/alcideio/rbac-tool/pkg/kube"
)

func NewCommandGenerateClusterRole() *cobra.Command {

	clusterContext := ""
	name := "custom-cluster-reader"
	namespace := "myappnamespace"
	generateKind := ""
	allowedGroups := []string{}
	//expandGroups := []string{}
	allowedVerb := []string{}
	denyResources := []string{}
	annotations := map[string]string{}

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
			kubeClient, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("Failed to create kubernetes client - %v", err)
			}

			computedPolicyRules, err := generateRules(generateKind, kubeClient.ServerPreferredResources, sets.NewString(denyResources...), sets.NewString(allowedGroups...), sets.NewString(allowedVerb...))
			if err != nil {
				return err
			}

			obj, err := generateRole(generateKind, computedPolicyRules, name, namespace, annotations)
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stdout, obj)

			return nil
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&generateKind, "generated-type", "t", "ClusterRole", "Role or ClusterRole")
	flags.StringVarP(&clusterContext, "cluster-context", "c", "", "Cluster.use 'kubectl config get-contexts' to list available contexts")
	flags.StringVar(&name, "name", "", "Name of Role/ClusterRole")
	flags.StringVarP(&namespace, "namespace", "n", "", "Namespace of Role/ClusterRole")
	//flags.StringSliceVarP(&expandGroups, "expand-groups", "g", []string{""},  "Comma separated list of API groups we would like to list all resource kinds rather than using wild cards '*'")
	flags.StringSliceVar(&allowedGroups, "allowed-groups", []string{"*"}, "Comma separated list of API groups we would like to allow '*'")
	flags.StringSliceVar(&allowedVerb, "allowed-verbs", []string{"*"}, "Comma separated list of verbs to include. To include all use '*'")
	flags.StringSliceVar(&denyResources, "deny-resources", []string{""}, "Comma separated list of resource.group - for example secret. to deny secret (core group) access")
	flags.StringToStringVar(&annotations, "annotations", map[string]string{}, "Custom annotations")

	return cmd
}

func generateRole(generateKind string, rules []rbacv1.PolicyRule, name string, namespace string, annotations map[string]string) (string, error) {
	var obj runtime.Object

	if generateKind == "ClusterRole" {
		obj = &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterRole",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Rules: rules,
		}
	} else {
		obj = &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Role",
				APIVersion: "rbac.authorization.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        name,
				Namespace:   namespace,
				Annotations: annotations,
			},
			Rules: rules,
		}
	}

	serializer := k8sJson.NewSerializerWithOptions(k8sJson.DefaultMetaFactory, nil, nil, k8sJson.SerializerOptions{Yaml: true, Pretty: true, Strict: true})
	var writer = bytes.NewBufferString("")
	err := serializer.Encode(obj, writer)
	if err != nil {
		return "", err
	}

	return writer.String(), nil
}

func generateRules(generateKind string, apiresourceList []*metav1.APIResourceList, denyResources sets.String, includeGroups sets.String, allowedVerbs sets.String) ([]rbacv1.PolicyRule, error) {
	isRole := generateKind == "Role"
	errs := []error{}

	computedPolicyRules := make([]rbacv1.PolicyRule, 0)

	//processedGroups := sets.NewString()

	for _, apiGroup := range apiresourceList {

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

			if isRole && !kind.Namespaced {
				//When generating role - non-namespaced resources are not relevant
				continue
			}

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

		if len(resourceList) == 0 || uniqueVerbs.Len() == 0 {
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
