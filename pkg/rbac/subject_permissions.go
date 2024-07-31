package rbac

import (
	"sort"
	"strings"

	"github.com/kylelemons/godebug/pretty"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/klog"
)

type PolicyRule struct {
	v1.PolicyRule

	//Specify the Roles or ClusterRoles this rule originated from
	OriginatedFrom []v1.RoleRef
}

type SubjectPermissions struct {
	Subject v1.Subject

	//Rules Per Namespace ... "" means cluster-wide
	Rules map[string][]PolicyRule
}

func NewSubjectPermissions(perms *Permissions) []SubjectPermissions {
	subjects := map[string]*SubjectPermissions{}

	for _, bindings := range perms.RoleBindings {
		for _, binding := range bindings {
			for _, subject := range binding.Subjects {
				var exist bool
				var subPerms *SubjectPermissions

				ns := binding.Namespace
				if strings.ToLower(binding.RoleRef.Kind) == "clusterrole" {
					ns = ""
				}

				if subject.Namespace == "" && subject.Kind == v1.ServiceAccountKind && binding.Namespace != "" {
					//If for some reason the namespace is absent from the subject for ServiceAccount - fill it
					subject.Namespace = binding.Namespace
				}

				roles, exist := perms.Roles[ns]
				if !exist {
					klog.V(6).Infof("[%v] %+v didn't find roles for namespace '%v'", subject.String(), binding, ns)
					continue
				}

				role, exist := roles[binding.RoleRef.Name]
				if !exist {
					klog.V(6).Infof("[%v] %+v didn't find role '%v' in '%v'", subject.String(), binding, binding.RoleRef.Name, ns)
					continue
				}

				sub := subject.String()
				subPerms, exist = subjects[sub]

				if !exist {
					subPerms = &SubjectPermissions{
						Subject: subject,
						Rules:   map[string][]PolicyRule{},
					}

					klog.V(6).Infof("[%v] %+v -- CREATE --", subject.String(), subject)
				}

				rules, exist := subPerms.Rules[binding.Namespace]
				if !exist {
					rules = []PolicyRule{}
				}

				roleRules := make([]PolicyRule, len(role.Rules))
				for i, _ := range role.Rules {
					roleRules[i].PolicyRule = role.Rules[i]
					roleRules[i].OriginatedFrom = []v1.RoleRef{binding.RoleRef}
				}

				klog.V(6).Infof("[%v] %+v -- UPDATE -- %v %v %+v", subject.String(), subject, len(rules), len(role.Rules), roleRules)

				rules = append(rules, roleRules...)
				subPerms.Rules[binding.Namespace] = rules
				subjects[sub] = subPerms
			}
		}
	}

	res := []SubjectPermissions{}
	for _, v := range subjects {
		res = append(res, *v)
	}

	klog.V(10).Infof("%v", pretty.Sprint(res))

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

	//The Role/ClusterRole rule references
	OriginatedFrom []v1.RoleRef `json:"originatedFrom,omitempty"`
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

					if len(rule.NonResourceURLs) == 0 {
						// The common case ... let's flatten the rule
						for _, apiGroup := range rule.APIGroups {
							for _, resource := range rule.Resources {
								subjectPolicy := NamespacedPolicyRule{
									Namespace:       namespace,
									Verb:            verb,
									APIGroup:        apiGroup,
									Resource:        resource,
									ResourceNames:   rule.ResourceNames,
									NonResourceURLs: rule.NonResourceURLs,
									OriginatedFrom:  rule.OriginatedFrom,
								}

								nsrules = append(nsrules, subjectPolicy)
							}

						}
					} else {
						// NonResourceURL ... not namespaced
						subjectPolicy := NamespacedPolicyRule{
							Namespace:       namespace,
							Verb:            verb,
							NonResourceURLs: rule.NonResourceURLs,
							OriginatedFrom:  rule.OriginatedFrom,
						}

						nsrules = append(nsrules, subjectPolicy)
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
