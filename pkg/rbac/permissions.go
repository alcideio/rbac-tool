package rbac

import (
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"

	"github.com/alcideio/rbac-tool/pkg/kube"
)

type Permissions struct {
	ServiceAccounts map[string]map[string]v1.ServiceAccount

	// Roles & RoleBinding maps captures Cluster & ClusterRoleBinding in namespace ""
	// - ClusterRoles are stored in Roles[""]
	// - ClusterRoleBindings are stored in RoleBindings[""]
	Roles        map[string]map[string]rbacv1.Role
	RoleBindings map[string]map[string]rbacv1.RoleBinding
}

func (p *Permissions) populateServiceAccounts(sas []v1.ServiceAccount) {
	for _, sa := range sas {

		if p.ServiceAccounts[sa.Namespace] == nil {
			p.ServiceAccounts[sa.Namespace] = make(map[string]v1.ServiceAccount)
		}

		p.ServiceAccounts[sa.Namespace][sa.Name] = sa

		klog.V(6).Infof("ServiceAccount %v/%v", sa.Namespace, sa.Name)
	}
}

func (p *Permissions) populateRoles(roles []rbacv1.Role) {
	for _, role := range roles {

		if p.Roles[role.Namespace] == nil {
			p.Roles[role.Namespace] = make(map[string]rbacv1.Role)
		}

		p.Roles[role.Namespace][role.Name] = role
		klog.V(6).Infof("Role %v/%v", role.Namespace, role.Name)
	}
}

func (p *Permissions) populateClusterRoles(clusterRoles []rbacv1.ClusterRole) {
	for _, role := range clusterRoles {
		if p.Roles[""] == nil {
			p.Roles[""] = make(map[string]rbacv1.Role)
		}

		aRole := rbacv1.Role{
			ObjectMeta: role.ObjectMeta,
			Rules:      role.Rules,
		}

		p.Roles[role.Namespace][role.Name] = aRole
		klog.V(6).Infof("ClusterRole %v", role.Name)
	}
}

func (p *Permissions) populateRoleBindings(bindings []rbacv1.RoleBinding) {
	for _, binding := range bindings {
		if p.RoleBindings[binding.Namespace] == nil {
			p.RoleBindings[binding.Namespace] = make(map[string]rbacv1.RoleBinding)
		}

		p.RoleBindings[binding.Namespace][binding.Name] = binding
		klog.V(6).Infof("RoleBinding %v/%v", binding.Namespace, binding.Name)
	}
}

func (p *Permissions) populateClusterRoleBindings(bindings []rbacv1.ClusterRoleBinding) {
	for _, binding := range bindings {
		if p.RoleBindings[""] == nil {
			p.RoleBindings[""] = make(map[string]rbacv1.RoleBinding)
		}

		aBindinig := rbacv1.RoleBinding{
			ObjectMeta: binding.ObjectMeta,
			Subjects:   binding.Subjects,
			RoleRef:    binding.RoleRef,
		}

		p.RoleBindings[""][binding.Name] = aBindinig
		klog.V(6).Infof("ClusterRoleBinding %v", aBindinig.Name)
	}
}

func NewPermissionsFromCluster(client *kube.KubeClient) (*Permissions, error) {
	permissions := &Permissions{}

	permissions.ServiceAccounts = make(map[string]map[string]v1.ServiceAccount)
	permissions.Roles = make(map[string]map[string]rbacv1.Role)
	permissions.RoleBindings = make(map[string]map[string]rbacv1.RoleBinding)

	sas, err := client.ListServiceAccounts(v1.NamespaceAll)
	if err != nil {
		return nil, err
	}

	permissions.populateServiceAccounts(sas)

	roles, err := client.ListRoles(v1.NamespaceAll)
	if err != nil {
		return nil, err
	}
	permissions.populateRoles(roles)

	clusterRoles, err := client.ListClusterRoles()
	if err != nil {
		return nil, err
	}
	permissions.populateClusterRoles(clusterRoles)

	bindings, err := client.ListRoleBindings(v1.NamespaceAll)
	if err != nil {
		return nil, err
	}
	permissions.populateRoleBindings(bindings)

	clusterBindings, err := client.ListClusterRoleBindings()
	if err != nil {
		return nil, err
	}
	permissions.populateClusterRoleBindings(clusterBindings)

	return permissions, nil
}

func NewPermissionsFromResourceList(objs []runtime.Object) (*Permissions, error) {
	permissions := &Permissions{}

	permissions.ServiceAccounts = make(map[string]map[string]v1.ServiceAccount)
	permissions.Roles = make(map[string]map[string]rbacv1.Role)
	permissions.RoleBindings = make(map[string]map[string]rbacv1.RoleBinding)

	sas := []v1.ServiceAccount{}
	roles := []rbacv1.Role{}
	clusterRoles := []rbacv1.ClusterRole{}
	bindings := []rbacv1.RoleBinding{}
	clusterBindings := []rbacv1.ClusterRoleBinding{}

	for _, obj := range objs {

		switch o := obj.(type) {
		case *v1.ServiceAccount:
			sas = append(sas, *o)
		case *rbacv1.Role:
			roles = append(roles, *o)
		case *rbacv1.ClusterRole:
			clusterRoles = append(clusterRoles, *o)
		case *rbacv1.RoleBinding:
			bindings = append(bindings, *o)
		case *rbacv1.ClusterRoleBinding:
			clusterBindings = append(clusterBindings, *o)

		default:
			klog.V(6).Infof("Skipping type %v", obj.GetObjectKind().GroupVersionKind().String())
		}
	}

	permissions.populateServiceAccounts(sas)
	permissions.populateRoles(roles)
	permissions.populateClusterRoles(clusterRoles)
	permissions.populateRoleBindings(bindings)
	permissions.populateClusterRoleBindings(clusterBindings)

	return permissions, nil
}
