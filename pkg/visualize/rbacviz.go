package visualize

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/emicklei/dot"
	"github.com/fatih/color"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"
	"github.com/alcideio/rbac-tool/pkg/utils"
)

func CreateRBACGraph(opts *Opts) error {
	inNs, exNs := utils.GetNamespaceSets(opts.IncludedNamespaces, opts.ExcludedNamespaces)

	rbacViz := RbacViz{
		opts: opts,

		includedNamespace: inNs,
		excludedNamespace: exNs,
	}

	err := rbacViz.initialize(opts)
	if err != nil {
		return err
	}

	g := rbacViz.renderGraph()

	legend := GraphLegend()

	utils.ConsolePrinter(fmt.Sprintf("Generating Graph and Saving as '%v'", color.HiBlueString(opts.Outfile)))

	return GenerateOutput(opts.Outfile, opts.Outformat, g, legend, opts)
}

type RbacViz struct {
	opts *Opts

	includedNamespace sets.String
	excludedNamespace sets.String
	permissions       Permissions

	includeSubjectsRegex *regexp.Regexp
}

func (r *RbacViz) initialize(opts *Opts) error {

	if opts.Infile == "" {
		//Connect to the cluster

		var client *kube.KubeClient
		var err error

		utils.ConsolePrinter(fmt.Sprintf("Connecting to cluster '%v'", color.HiBlueString(opts.ClusterContext)))

		client, err = kube.NewClient(opts.ClusterContext)
		if err != nil {
			return fmt.Errorf("Failed to create kubernetes client - %v", err)
		}

		perms, err := rbac.NewPermissionsFromCluster(client)
		if err != nil {
			return err
		}

		r.permissions.Permissions = *perms
		r.permissions.Pods = make(map[string]map[string]v1.Pod)
		r.permissions.ServiceAccountsUsed = sets.NewString()

		if opts.ShowPodsOnly {
			pods, err := client.ListPods(v1.NamespaceAll)
			if err != nil {
				return err
			}

			for _, pod := range pods {
				if r.permissions.Pods[pod.Namespace] == nil {
					r.permissions.Pods[pod.Namespace] = make(map[string]v1.Pod)
				}

				r.permissions.Pods[pod.Namespace][pod.Name] = pod

				r.permissions.ServiceAccountsUsed.Insert(fmt.Sprintf("%s/%s", pod.Namespace, pod.Spec.ServiceAccountName))
				klog.V(6).Infof("Pod %v/%v use ServiceAccount %v/%v", pod.Namespace, pod.Name, pod.Namespace, pod.Spec.ServiceAccountName)
			}
		}

	} else {
		utils.ConsolePrinter(fmt.Sprintf("Loading Resources from '%v'", color.HiBlueString(opts.Infile)))

		// Load from file/stdin
		objs, err := utils.ReadObjectsFromFile(opts.Infile)
		if err != nil {
			return err
		}

		klog.V(5).Infof("Loaded %v resources", len(objs))

		perms, err := rbac.NewPermissionsFromResourceList(objs)
		if err != nil {
			return err
		}

		r.permissions.Permissions = *perms
		r.permissions.Pods = make(map[string]map[string]v1.Pod)
		r.permissions.ServiceAccountsUsed = sets.NewString()
	}

	var err error
	r.includeSubjectsRegex, err = regexp.Compile(opts.IncludeSubjectsRegex)

	if err != nil {
		return err
	}

	return nil
}

func (r *RbacViz) isBindingUsed(binding rbacv1.RoleBinding) bool {
	klog.V(5).Infof(">>> [process][ClusterRole/Role Binding %v/%v]", binding.Namespace, binding.Name)
	for _, subject := range binding.Subjects {

		if !utils.IsNamespaceIncluded(subject.Namespace, r.includedNamespace, r.excludedNamespace) {
			klog.V(5).Infof("\t\t>>> [skip][ClusterRole/Role Binding %v/%v] ServiceAccount %v/%v - not in namespace inclusion list", binding.Namespace, binding.Name, subject.Namespace, subject.Name)
			continue
		}

		if !r.includeSubjectsRegex.MatchString(subject.Name) {
			klog.V(5).Infof("\t\t>>> [skip][Subject] ServiceAccount %v/%v - Subject '%v' does NOT match the regexp '%v'", binding.Namespace, binding.Name, subject.Name, r.includeSubjectsRegex.String())
			continue
		}

		if !r.opts.ShowPodsOnly {
			klog.V(5).Infof("\t\t>> [used][ClusterRole/Role Binding %v/%v] - used by %v/%v", binding.Namespace, binding.Name, binding.Namespace, subject.Name)
			return true
		}

		if r.permissions.ServiceAccountsUsed.Has(fmt.Sprintf("%s/%s", subject.Namespace, subject.Name)) {
			klog.V(5).Infof("\t\t>> [used][ClusterRole/Role Binding %v/%v] - used by %v/%v", binding.Namespace, binding.Name, subject.Namespace, subject.Name)
			return true
		}

		klog.V(5).Infof("\t\t>>> [skip][ServiceAccount %v/%v] not used", subject.Namespace, subject.Name)
	}

	klog.V(5).Infof("<<< [skip][ClusterRole/Role Binding %v/%v]", binding.Namespace, binding.Name)
	return false
}

func GraphLegend() *dot.Graph {
	g := newGraph()
	renderLegend(g)

	return g
}

func (r *RbacViz) renderGraph() *dot.Graph {
	g := newGraph()

	if r.opts.ShowLegend {
		renderLegend(g)
	}

	for _, bindings := range r.permissions.RoleBindings {
		for _, binding := range bindings {

			// Check that this a namespace we would like to visualize
			// Binding with "" namespace are clusterrole bindings
			if binding.Namespace != "" && !utils.IsNamespaceIncluded(binding.Namespace, r.includedNamespace, r.excludedNamespace) {
				klog.V(5).Infof("Skiping %v/%v - namespace '%s' not included", binding.Namespace, binding.Name, binding.Namespace)
				continue
			}

			//Lets skip bindings that do not point something active (TODO: add option to visualize this use case as well)
			if !r.isBindingUsed(binding) {
				klog.V(5).Infof("Binding %v/%v not used by any service account", binding.Namespace, binding.Name)
				continue
			}

			//
			//  [Pod] --> [ServiceAccount]<----[Binding]--->[Role]
			//
			nsSubGraph := newNamespaceSubgraph(g, binding.Namespace)

			bindingNode := r.newBindingNode(nsSubGraph, binding)
			roleNode, _ := r.newRoleAndRulesNodePair(nsSubGraph, binding.Namespace, binding.RoleRef)

			newBindingToRoleEdge(bindingNode, roleNode)

			saNodes := []dot.Node{}
			for _, subject := range binding.Subjects {
				if !r.includeSubjectsRegex.MatchString(subject.Name) {
					klog.V(5).Infof("\t\t>>> [skip][Subject] ServiceAccount %v/%v - Subject '%v' does NOT match the regexp '%v'", binding.Namespace, binding.Name, subject.Name, r.includeSubjectsRegex.String())
					continue
				}
				gns := newNamespaceSubgraph(g, subject.Namespace)
				subjectNode := r.newSubjectNode(gns, subject.Kind, subject.Namespace, subject.Name)
				saNodes = append(saNodes, subjectNode)
			}

			for _, saNode := range saNodes {
				newSubjectToBindingEdge(saNode, bindingNode)
			}
		}
	}

	return g
}

func renderLegend(g *dot.Graph) {
	legend := g.Subgraph("Legend", dot.ClusterOption{})
	legend.Attr("style", "invis")
	namespace := newNamespaceSubgraph(legend, "Namespace")

	sa := newSubjectNode0(namespace, "Kind", "Subject", true, false)
	missingSa := newSubjectNode0(namespace, "Kind", "Missing Subject", false, false)

	role := newRoleNode(namespace, "ns", "Role", true, false)
	clusterRoleBoundLocally := newClusterRoleNode(namespace, "ns", "ClusterRole", true, false) // bound by (namespaced!) RoleBinding
	clusterrole := newClusterRoleNode(legend, "", "ClusterRole", true, false)

	roleBinding := newRoleBindingNode(namespace, "RoleBinding", false)
	newSubjectToBindingEdge(sa, roleBinding)
	newSubjectToBindingEdge(missingSa, roleBinding)
	newBindingToRoleEdge(roleBinding, role)

	roleBinding2 := newRoleBindingNode(namespace, "RoleBinding-to-ClusterRole", false)
	roleBinding2.Attr("label", "RoleBinding")
	newSubjectToBindingEdge(sa, roleBinding2)
	newBindingToRoleEdge(roleBinding2, clusterRoleBoundLocally)

	clusterRoleBinding := newClusterRoleBindingNode(legend, "ClusterRoleBinding", false)
	newSubjectToBindingEdge(sa, clusterRoleBinding)
	newBindingToRoleEdge(clusterRoleBinding, clusterrole)

	nsrules := newRulesNode0(namespace, "ns", "Role", "Namespace-scoped access rules", false)
	newRoleToRulesEdge(role, nsrules)

	nsrules2 := newRulesNode0(namespace, "ns", "ClusterRole", "Namespace-scoped access rules From ClusterRole", false)
	//nsrules2.Attr("label", "Namespace-scoped\naccess rules")
	newRoleToRulesEdge(clusterRoleBoundLocally, nsrules2)

	clusterrules := newRulesNode0(legend, "", "ClusterRole", "Cluster-scoped access rules", false)
	newRoleToRulesEdge(clusterrole, clusterrules)

}

func (r *RbacViz) newBindingNode(gns *dot.Graph, binding rbacv1.RoleBinding) dot.Node {
	if binding.Namespace == "" {
		return newClusterRoleBindingNode(gns, binding.Name, false)
	} else {
		return newRoleBindingNode(gns, binding.Name, false)
	}
}

func (r *RbacViz) newRoleAndRulesNodePair(gns *dot.Graph, bindingNamespace string, roleRef rbacv1.RoleRef) (dot.Node, *dot.Node) {
	var roleNode dot.Node
	var rulesNode *dot.Node
	var roleNamespace string

	if roleRef.Kind == "ClusterRole" {
		roleNamespace = ""
		roleNode = newClusterRoleNode(gns, bindingNamespace, roleRef.Name, r.roleExists("", roleRef.Name), false)
	} else {
		roleNamespace = bindingNamespace
		roleNode = newRoleNode(gns, bindingNamespace, roleRef.Name, r.roleExists(bindingNamespace, roleRef.Name), false)
	}

	if r.opts.ShowRules {
		rulesNode = r.newFormattedRulesNode(gns, roleNamespace, roleRef.Name, false)
		if rulesNode != nil {
			newRoleToRulesEdge(roleNode, *rulesNode)
		}
	}
	return roleNode, rulesNode
}

func (r *RbacViz) roleExists(roleNamespace string, roleName string) bool {
	if roles, nsExists := r.permissions.Roles[roleNamespace]; nsExists {
		if _, roleExists := roles[roleName]; roleExists {
			return true
		}
	}
	return false
}

func (r *RbacViz) newSubjectNode(gns *dot.Graph, kind string, ns string, name string) dot.Node {
	return newSubjectNode0(gns, kind, name, r.subjectExists(kind, ns, name), false)
}

func (r *RbacViz) subjectExists(kind string, ns string, name string) bool {
	if strings.ToLower(kind) != strings.ToLower("ServiceAccount") {
		klog.V(5).Infof("[Subject Exist] %v/%v (%v) exist", ns, name, kind)
		return true // assume users and groups exist
	}

	if sas, nsExists := r.permissions.ServiceAccounts[ns]; nsExists {
		if _, saExists := sas[name]; saExists {
			return true
		} else {
			klog.V(5).Infof("[Subject Does Not Exist] %v/%v (%v) exist - Couldn't find service account in Namespace %v", ns, name, kind, ns)
		}
	} else {
		klog.V(5).Infof("[Subject Does Not Exist] %v/%v (%v) exist - Couldn't find namespace '%v' with service accounts", ns, name, kind, ns)
	}

	return false
}

func (r *RbacViz) newFormattedRulesNode(g *dot.Graph, namespace, roleName string, highlight bool) *dot.Node {
	var rulesText string

	table := `	
		<table border="0" align="left">
		  <tr>
			<td align="left" border="1" sides="b">ApiGroup</td>
			<td align="left" border="1" sides="b">Kind</td>
			<td align="left" border="1" sides="b">Names</td>
			<td align="left" border="1" sides="b">Verbs</td>
			<td align="left" border="1" sides="b">NonResourceURI</td>
		  </tr>
		  %s
		</table>
`
	if roles, found := r.permissions.Roles[namespace]; found {
		if role, found := roles[roleName]; found {
			for _, rule := range role.Rules {
				rulesText += toRuleToTableRow(rule)
			}
		}
	}

	if rulesText == "" {
		return nil
	}

	rulesText = fmt.Sprintf(table, rulesText)

	node := newRulesNode0(g, namespace, roleName, rulesText, highlight)
	return &node
}

func toRuleToTableRow(rule rbacv1.PolicyRule) string {
	verbs := strings.Join(rule.Verbs, ",")
	apiGroups := ""
	resources := ""
	resourcesNames := ""
	nonResourceURLs := ""

	if len(rule.Resources) > 0 {
		resources += strings.Join(rule.Resources, ",")
	}
	if len(rule.ResourceNames) > 0 {
		resourcesNames += strings.Join(rule.ResourceNames, ",")
	} else {
		resourcesNames = "*"
	}

	if len(rule.NonResourceURLs) > 0 {
		nonResourceURLs += strings.Join(rule.NonResourceURLs, ",")
	}

	for i, apigroup := range rule.APIGroups {
		if apigroup == "" {
			apigroup = "core"
		}

		if i > 0 {
			apiGroups = fmt.Sprint(apigroup, ",", apiGroups)
		} else {
			apiGroups = apigroup
		}
	}

	tableRow := `
	  <tr>
		<td align="left">%s</td>
		<td align="left">%s</td>
		<td align="left">%s</td>
		<td align="left">%s</td>
		<td align="left">%s</td>
	  </tr>
`
	return fmt.Sprintf(tableRow, apiGroups, resources, resourcesNames, verbs, nonResourceURLs)
}
