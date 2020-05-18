package rbac

import (
	v1 "k8s.io/api/rbac/v1"
	"sort"
)

type SubjectPermissions struct {
	Subject v1.Subject

	//Rules Per Namespace ... "" means cluster-wide
	Rules map[string][]v1.PolicyRule
}

func NewSubjectPermissions(perms *Permissions) []SubjectPermissions {
	subjects := map[string]*SubjectPermissions{}

	for _, bindings := range perms.RoleBindings {
		for _, binding := range bindings {
			for _, subject := range binding.Subjects {
				var exist bool
				var subPerms *SubjectPermissions

				roles, exist := perms.Roles[binding.Namespace]
				if !exist {
					continue
				}

				role, exist := roles[binding.RoleRef.Name]
				if !exist {
					continue
				}

				sub := subject.String()
				subPerms, exist = subjects[sub]

				if !exist {
					subPerms = &SubjectPermissions{
						Subject: subject,
						Rules:   map[string][]v1.PolicyRule{},
					}
				}

				rules, exist := subPerms.Rules[binding.Namespace]
				if !exist {
					rules = []v1.PolicyRule{}
				}

				rules = append(rules, role.Rules...)
				subPerms.Rules[binding.Namespace] = rules
				subjects[sub] = subPerms
			}
		}
	}

	res := []SubjectPermissions{}
	for _, v := range subjects {
		res = append(res, *v)
	}

	return res
}

func ReplaceToWildCard(l []string) {
	for i, _ := range l {
		if l[i] == "" {
			l[i] = "*"
		}
	}
}

func ReplaceToCore(l []string) {
	for i, _ := range l {
		if l[i] == "" {
			l[i] = "core"
		}
	}
}

type NamespacedPolicyRule struct {
	Namespace string `json:"namespace,omitempty"`

	// Verbs is a list of Verbs that apply to ALL the ResourceKinds and AttributeRestrictions contained in this rule.  VerbAll represents all kinds.
	Verb string `json:"verb"`

	// The name of the APIGroup that contains the resources.
	APIGroup string `json:"apiGroup,omitempty"`

	// Resources is a list of resources this rule applies to.  ResourceAll represents all resources.
	Resource string `json:"resource,omitempty"`

	// ResourceNames is an optional white list of names that the rule applies to.  An empty set means that everything is allowed.
	ResourceNames []string `json:"resourceNames,omitempty"`

	// NonResourceURLs is a set of partial urls that a user should have access to.  *s are allowed, but only as the full, final step in the path
	// Since non-resource URLs are not namespaced, this field is only applicable for ClusterRoles referenced from a ClusterRoleBinding.
	NonResourceURLs []string `json:"nonResourceURLs,omitempty"`
}

type SubjectPolicyList struct {
	v1.Subject

	AllowedTo []NamespacedPolicyRule `json:"allowedTo,omitempty"`
}

func NewSubjectPermissionsList(policies []SubjectPermissions) []SubjectPolicyList {
	subjectPolicyList := []SubjectPolicyList{}

	for _, p := range policies {
		nsrules := []NamespacedPolicyRule{}
		for namespace, rules := range p.Rules {
			if namespace == "" {
				namespace = "*"
			}

			for _, rule := range rules {
				//Normalize the strings
				ReplaceToCore(rule.APIGroups)
				ReplaceToWildCard(rule.Resources)
				ReplaceToWildCard(rule.ResourceNames)
				ReplaceToWildCard(rule.Verbs)
				ReplaceToWildCard(rule.NonResourceURLs)

				sort.Strings(rule.APIGroups)
				sort.Strings(rule.Resources)
				sort.Strings(rule.ResourceNames)
				sort.Strings(rule.Verbs)
				sort.Strings(rule.NonResourceURLs)

				for _, verb := range rule.Verbs {
					for _, apiGroup := range rule.APIGroups {
						for _, resource := range rule.Resources {
							//if len(rule.APIGroups) == 0 {
							//	rule.APIGroups = []string{"-"}
							//}
							//
							//if len(rule.Resources) == 0 {
							//	rule.Resources = []string{"-"}
							//}
							//
							//if len(rule.ResourceNames) == 0 {
							//	if len(rule.APIGroups) == 0 {
							//		rule.ResourceNames = []string{"-"}
							//	} else {
							//		rule.ResourceNames = []string{"-"}
							//	}
							//}
							//
							//if len(rule.NonResourceURLs) == 0 {
							//	rule.NonResourceURLs = []string{"-"}
							//}

							subjectPolicy := NamespacedPolicyRule{
								Namespace:       namespace,
								Verb:            verb,
								APIGroup:        apiGroup,
								Resource:        resource,
								ResourceNames:   rule.ResourceNames,
								NonResourceURLs: rule.NonResourceURLs,
							}

							nsrules = append(nsrules, subjectPolicy)
						}

					}
				}
			}
		}
		subjectPolicyList = append(
			subjectPolicyList, SubjectPolicyList{
				Subject:   p.Subject,
				AllowedTo: nsrules,
			})
	}

	return subjectPolicyList
}
