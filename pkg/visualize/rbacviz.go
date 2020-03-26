package visualize

import (
	"fmt"
	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/utils"
	"github.com/fatih/color"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"strings"

	"github.com/emicklei/dot"
)

func CreateRBACGraph(client *kube.KubeClient, opts *Opts) error {

	inNs, exNs := utils.GetNamespaceSets(opts.IncludedNamespaces, opts.ExcludedNamespaces)

	rbacViz := RbacViz{
		opts: *opts,
		client: client,

		includedNamespace: inNs,
		excludedNamespace: exNs,

	}

	err := rbacViz.initialize()
	if err != nil {
		return err
	}

	g := rbacViz.renderGraph()

	legend := GraphLegend()

	utils.ConsolePrinter(fmt.Sprintf("Generating Graph and Saving as '%v'", color.HiBlueString(opts.Outfile)))

	return GenerateOutput(opts.Outfile, opts.Outformat, g, legend, opts)
}

type RbacViz struct {
	opts        Opts
	client *kube.KubeClient

	includedNamespace sets.String
	excludedNamespace sets.String
	permissions Permissions
}

func (r *RbacViz) initialize() (err error) {

	r.permissions.ServiceAccounts = make(map[string]map[string]v1.ServiceAccount)
	r.permissions.Roles = make(map[string]map[string]rbacv1.Role)
	r.permissions.RoleBindings = make(map[string]map[string]rbacv1.RoleBinding)
	r.permissions.Pods = make(map[string]map[string]v1.Pod)
	r.permissions.ServiceAccountsUsed = sets.NewString()

	sas, err := r.client.ListServiceAccounts(v1.NamespaceAll)
	if err != nil {
		return err
	}

	for _,sa := range sas {

		if r.permissions.ServiceAccounts[sa.Namespace] == nil {
			r.permissions.ServiceAccounts[sa.Namespace] = make(map[string]v1.ServiceAccount)
		}

		r.permissions.ServiceAccounts[sa.Namespace][sa.Name] = sa

		klog.V(6).Infof("ServiceAccount %v/%v", sa.Namespace, sa.Name)
	}

	roles, err := r.client.ListRoles(v1.NamespaceAll)
	if err != nil {
		return err
	}

	for _,role := range roles {

		if r.permissions.Roles[role.Namespace] == nil {
			r.permissions.Roles[role.Namespace] = make(map[string]rbacv1.Role)
		}

		r.permissions.Roles[role.Namespace][role.Name] = role
		klog.V(6).Infof("Role %v/%v", role.Namespace, role.Name)
	}

	clusterRoles, err := r.client.ListClusterRoles()
	if err != nil {
		return err
	}

	for _,role := range clusterRoles {
		if r.permissions.Roles[""] == nil {
			r.permissions.Roles[""] = make(map[string]rbacv1.Role)
		}

		aRole := rbacv1.Role{
			ObjectMeta: role.ObjectMeta,
			Rules:      role.Rules,
		}

		r.permissions.Roles[role.Namespace][role.Name] = aRole
		klog.V(6).Infof("ClusterRole %v", role.Name)
	}

	bindings, err := r.client.ListRoleBindings(v1.NamespaceAll)
	if err != nil {
		return err
	}

	for _,binding := range bindings {
		if r.permissions.RoleBindings[binding.Namespace] == nil {
			r.permissions.RoleBindings[binding.Namespace] = make(map[string]rbacv1.RoleBinding)
		}

		r.permissions.RoleBindings[binding.Namespace][binding.Name] = binding
		klog.V(6).Infof("RoleBinding %v/%v", binding.Namespace, binding.Name)
	}

	clusterBindings, err := r.client.ListClusterRoleBindings()
	if err != nil {
		return err
	}

	for _,binding := range clusterBindings {
		if r.permissions.RoleBindings[""] == nil {
			r.permissions.RoleBindings[""] = make(map[string]rbacv1.RoleBinding)
		}

		aBindinig := rbacv1.RoleBinding{
			ObjectMeta: binding.ObjectMeta,
			Subjects:   binding.Subjects,
			RoleRef:    binding.RoleRef,
		}

		r.permissions.RoleBindings[""][binding.Name] = aBindinig
		klog.V(6).Infof("ClusterRoleBinding %v", aBindinig.Name)
	}

	pods, err := r.client.ListPods(v1.NamespaceAll)
	if err != nil {
		return err
	}

	for _,pod := range pods {
		if r.permissions.Pods[pod.Namespace] == nil {
			r.permissions.Pods[pod.Namespace] = make(map[string]v1.Pod)
		}

		r.permissions.Pods[pod.Namespace][pod.Name] = pod

		r.permissions.ServiceAccountsUsed.Insert(fmt.Sprintf("%s/%s", pod.Namespace, pod.Spec.ServiceAccountName))
		klog.V(6).Infof("Pod %v/%v use ServiceAccount %v/%v", pod.Namespace, pod.Name, pod.Namespace, pod.Spec.ServiceAccountName)
	}

	return nil
}

func (r *RbacViz) isBindingUsed(binding rbacv1.RoleBinding) bool {
	for _,subject := range binding.Subjects {

		if !utils.IsNamespaceIncluded(subject.Namespace, r.includedNamespace, r.excludedNamespace) {
			klog.V(5).Infof(">>> [skip][ClusterRole/Role Binding %v/%v] ServiceAccount %v/%v - not in namespace inclusion list", binding.Namespace, binding.Name, subject.Namespace, subject.Name)
			continue
		}


		if r.permissions.ServiceAccountsUsed.Has(fmt.Sprintf("%s/%s", subject.Namespace, subject.Name)) {
			klog.V(5).Infof(">> [used][ClusterRole/Role Binding %v/%v] - used by %v/%v", binding.Namespace, binding.Name, subject.Namespace, subject.Name)
			return true
		}

		klog.V(5).Infof(">>> [skip][ServiceAccount %v/%v] not used", subject.Namespace, subject.Name)
	}

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

			//Check that this a namespace we would like to visualize
			if !utils.IsNamespaceIncluded(binding.Namespace, r.includedNamespace, r.excludedNamespace) {
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
			roleNode := r.newRoleAndRulesNodePair(nsSubGraph, binding.Namespace, binding.RoleRef)

			newBindingToRoleEdge(bindingNode, roleNode)

			saNodes := []dot.Node{}
			for _, subject := range binding.Subjects {
				gns := newNamespaceSubgraph(g, subject.Namespace)
				subjectNode := r.newSubjectNode(gns, subject.Kind, subject.Namespace, subject.Name)
				saNodes = append(saNodes, subjectNode)
			}

			for _, saNode := range saNodes {
				newSubjectToBindingEdge(saNode, bindingNode)
			}
		}
	}

	//for ns, sas := range r.permissions.ServiceAccounts {
	//	if !r.namespaceSelected(ns) {
	//		continue
	//	}
	//	gns := newNamespaceSubgraph(g, ns)
	//
	//	for sa, _ := range sas {
	//		renderSA := r.opts.resourceKind == "" || (r.namespaceSelected(ns) && r.resourceNameSelected(sa))
	//		if renderSA {
	//			r.newSubjectNode(gns, "ServiceAccount", ns, sa)
	//		}
	//	}
	//}

	// draw any additional Roles that weren't referenced by bindings (and thus already drawn)
	//for ns, roles := range r.permissions.Roles {
	//	var renderRoles bool
	//
	//	areClusterRoles := ns == ""
	//	if areClusterRoles {
	//		renderRoles = (r.opts.resourceKind == "" || r.opts.resourceKind == kindClusterRole) && r.allNamespaces()
	//	} else {
	//		renderRoles = (r.opts.resourceKind == "" || r.opts.resourceKind == kindRole) && r.namespaceSelected(ns)
	//	}
	//
	//	if !renderRoles {
	//		continue
	//	}
	//
	//	nsSubGraph := newNamespaceSubgraph(g, ns)
	//	for roleName, _ := range roles {
	//		if utils.IsNamespaceIncluded(ns, r.includedNamespace, r.excludedNamespace) &&  r.resourceNameSelected(roleName) {
	//			r.newRoleAndRulesNodePair(nsSubGraph, "", NamespacedName{ns, roleName})
	//		}
	//	}
	//}

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
	nsrules2.Attr("label", "Namespace-scoped\naccess rules")
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

func (r *RbacViz) newRoleAndRulesNodePair(gns *dot.Graph, bindingNamespace string, roleRef rbacv1.RoleRef) dot.Node {
	var roleNode dot.Node
	var roleNamespace string

	if roleRef.Kind == "ClusterRole" {
		roleNamespace = ""
		roleNode = newClusterRoleNode(gns, bindingNamespace, roleRef.Name, r.roleExists("", roleRef.Name), false)
	} else {
		roleNamespace = bindingNamespace
		roleNode = newRoleNode(gns, bindingNamespace, roleRef.Name, r.roleExists(bindingNamespace, roleRef.Name), false)
	}
	
	if r.opts.ShowRules {
		rulesNode := r.newFormattedRulesNode(gns, roleNamespace, roleRef.Name, false)
		if rulesNode != nil {
			newRoleToRulesEdge(roleNode, *rulesNode)
		}
	}
	return roleNode
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
		return true // assume users and groups exist
	}

	if sas, nsExists := r.permissions.ServiceAccounts[ns]; nsExists {
		if _, saExists := sas[name]; saExists {
			return true
		}
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

	for i,apigroup := range rule.APIGroups {
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