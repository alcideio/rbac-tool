package visualize

import (
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Opts struct {
	ClusterContext string
	Outfile        string
	Outformat        string
	ShowRules      bool
	ShowLegend     bool

	IncludedNamespaces string
	ExcludedNamespaces string

	resourceKind  string
	resourceNames []string
}


type Permissions struct {
	ServiceAccounts map[string]map[string]v1.ServiceAccount  // map[namespace]map[name]ServiceAccount
	Roles           map[string]map[string]rbacv1.Role    // ClusterRoles are stored in Roles[""]
	RoleBindings    map[string]map[string]rbacv1.RoleBinding // ClusterRoleBindings are stored in RoleBindings[""]


	ServiceAccountsUsed sets.String
	Pods                map[string]map[string]v1.Pod //map[namespace]map[name]Pod
}
