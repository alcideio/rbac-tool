package analysis

import (
	"time"

	"github.com/google/uuid"
)

func DefaultAnalysisConfig() *AnalysisConfig {
	return &AnalysisConfig{
		AnalysisConfigInfo: AnalysisConfigInfo{
			Name:        "InsightCloudSec",
			Description: "Rapid7 InsightCloudSec default RBAC analysis rules",
			Uuid:        uuid.MustParse("9371719c-1031-468c-91ed-576fdc9e9f59").String(),
		},

		Rules: defaultRules,
		GlobalExclusions: []Exclusion{
			{
				Disabled:     false,
				Comment:      "Exclude kube-system from analysis",
				AddedBy:      "InsightCloudSec@rapid7.com",
				LastModified: time.Now().Format(time.RFC3339),
				SnoozeUntil:  0,
				Expression:   `has(subject.namespace) && (subject.namespace == "kube-system")`,
			},
			{
				Disabled:     false,
				Comment:      "Exclude system roles from analysis",
				AddedBy:      "InsightCloudSec@rapid7.com",
				LastModified: time.Now().Format(time.RFC3339),
				SnoozeUntil:  0,
				Expression:   `has(subject.name) && subject.name.startsWith('system:')`,
			},
		},
	}
}

func ExportDefaultConfig(format string) (string, error) {
	c := DefaultAnalysisConfig()

	return ExportAnalysisConfig(format, c)
}

var defaultRules Rules = []Rule{
	{Name: "Secret Readers",
		Description: "Capture principals that can read secrets",
		Recommendation: `
"Review the policy rules for \'" + (has(subject.namespace) ? subject.namespace +"/" : "") + subject.name + "\' ("+ subject.kind +") by running \'rbac-tool policy-rules -e " + subject.name +"\'" +
"\nYou can visualize the RBAC policy by running \'rbac-tool viz --include-subjects=" + subject.name +"\'"
`,
		Uuid:       uuid.MustParse("3c942117-f4ff-423a-83d4-f7d6b75a6b78").String(),
		Severity:   SEVERITY_HIGH,
		References: []string{},
		AnalysisExpr: `
				subjects.filter(
					subject, subject.allowedTo.exists(
						rule, 
						(has(rule.verb)     && rule.verb in ['get', '*'])         && 
						(has(rule.resource) && rule.resource in ['secrets', '*']) && 
                        (has(rule.apiGroup) && rule.apiGroup in ['core', '*'])
					)
				)`,
		Exclusions: []Exclusion{},
	},
	{Name: "Workload Creators",
		Description: "Capture principals that can create workloads of any kind (Deployments, Jobs, ...)",
		Recommendation: `
"Review the policy rules for \'" + (has(subject.namespace) ? subject.namespace +"/" : "") + subject.name + "\' ("+ subject.kind +") by running \'rbac-tool policy-rules -e " + subject.name +"\'" +
"\nYou can visualize the RBAC policy by running \'rbac-tool viz --include-subjects=" + subject.name +"\'"
`,
		Uuid:     uuid.MustParse("d5f5ea0c-82e9-4289-ba04-b40cc46be017").String(),
		Severity: SEVERITY_HIGH,
		References: []string{
			"https://www.impidio.com/blog/kubernetes-rbac-security-pitfalls",
		},
		AnalysisExpr: `
				subjects.filter(
					subject, subject.allowedTo.exists(
						rule,
						(has(rule.verb)     && rule.verb in ['create', '*'])         &&
						(has(rule.resource) && rule.resource in ['deployments', 'replicationcontrollers','daemonsets', 'statefulsets', 'replicasets', 'pods', 'jobs','cronjobs', '*']) &&
                       (has(rule.apiGroup) && rule.apiGroup in ['core', 'batch','*'])
					)
				)`,
		Exclusions: []Exclusion{},
	},
	{Name: "Identify Privileges Escalators - via impersonate",
		Description: "Capture principals that can escalate privileges through the use of impersonation",
		Recommendation: `
"Review the policy rules for \'" + (has(subject.namespace) ? subject.namespace +"/" : "") + subject.name + "\' ("+ subject.kind +") by running \'rbac-tool policy-rules -e " + subject.name +"\'" +
"\nYou can visualize the RBAC policy by running \'rbac-tool viz --include-subjects=" + subject.name +"\'"
`,
		Uuid:     uuid.MustParse("d5f5ea0c-82e9-4289-ba04-b40cc46be017").String(),
		Severity: SEVERITY_CRIT,
		References: []string{
			"https://www.impidio.com/blog/kubernetes-rbac-security-pitfalls",
			"https://kubernetes.io/docs/reference/access-authn-authz/rbac/",
		},
		AnalysisExpr: `
				subjects.filter(
					subject, subject.allowedTo.exists(
						rule,
						(has(rule.verb)     && rule.verb in ['impersonate', '*'])         &&
						(has(rule.resource) && rule.resource in ['users','groups','serviceaccounts', '*']) &&
                       (has(rule.apiGroup) && rule.apiGroup in ['core', '*'])
					)
				)`,
		Exclusions: []Exclusion{},
	},
	{Name: "Identify Privileges Escalators - via bind or escalate",
		Description: "Capture principals that can escalate privileges through the use of special API verbs 'bind' or 'escalate'",
		Recommendation: `
"Review the policy rules for \'" + (has(subject.namespace) ? subject.namespace +"/" : "") + subject.name + "\' ("+ subject.kind +") by running \'rbac-tool policy-rules -e " + subject.name +"\'" +
"\nYou can visualize the RBAC policy by running \'rbac-tool viz --include-subjects=" + subject.name +"\'"
`,
		Uuid:     uuid.MustParse("d5f5ea0c-82e9-4289-ba04-b40cc46be017").String(),
		Severity: SEVERITY_CRIT,
		References: []string{
			"https://www.impidio.com/blog/kubernetes-rbac-security-pitfalls",
			"https://kubernetes.io/docs/reference/access-authn-authz/rbac/",
		},
		AnalysisExpr: `
				subjects.filter(
					subject, subject.allowedTo.exists(
						rule,
						(has(rule.verb)     && rule.verb in ['create', '*'])         &&
						(has(rule.resource) && rule.resource in ['deployments', 'replicationcontrollers','daemonsets', 'statefulsets', 'replicasets', 'pods', 'jobs','cronjobs', '*']) &&
                       (has(rule.apiGroup) && rule.apiGroup in ['core', 'batch','*'])
					)
				)`,
		Exclusions: []Exclusion{},
	},
}
