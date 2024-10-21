package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

func NewCommandPolicyRules() *cobra.Command {

	clusterContext := ""
	regex := ""
	inverse := false
	output := "table"
	// Support overrides
	cmd := &cobra.Command{
		Use:     "policy-rules",
		Aliases: []string{"rules", "rule", "policy", "pr"},
		Short:   "RBAC List Policy Rules For subject (user/group/serviceaccount) name",
		Long: `
List Kubernetes RBAC policy rules for a given User/ServiceAccount/Group

Examples:

# Search All Service Accounts
rbac-tool policy-rules -e '.*'

# Search All Service Accounts that contain myname
rbac-tool policy-rules -e '.*myname.*'

# Lookup System Accounts (all accounts that start with system: )
rbac-tool policy-rules -e '^system:.*'

# Lookup all accounts that DO NOT start with system: )
rbac-tool policy-rules -ne '^system:.*'

# Leveraging jmespath for further filtering and implementing who-can
rbac-tool policy-rules -o json  | jp "[? @.allowedTo[? (verb=='get' || verb=='*') && (apiGroup=='core' || apiGroup=='*') && (resource=='secrets' || resource == '*')  ]].{name: name, namespace: namespace, kind: kind}"

`,
		Hidden: false,
		RunE: func(c *cobra.Command, args []string) error {
			var re *regexp.Regexp
			var err error

			if regex != "" {
				re, err = regexp.Compile(regex)
			} else {
				if len(args) != 1 {
					re, err = regexp.Compile(fmt.Sprintf(`.*`))
				} else {
					re, err = regexp.Compile(fmt.Sprintf(`(?mi)%v`, args[0]))
				}
			}

			if err != nil {
				return err
			}

			client, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("Failed to create kubernetes client - %v", err)
			}

			perms, err := rbac.NewPermissionsFromCluster(client)
			if err != nil {
				return err
			}

			policies := rbac.NewSubjectPermissions(perms)
			filteredPolicies := []rbac.SubjectPermissions{}
			for _, policy := range policies {
				match := re.MatchString(policy.Subject.Name)

				//  match    inverse
				//  -----------------
				//  true     true   --> skip
				//  true     false  --> keep
				//  false    true   --> keep
				//  false    false  --> skip
				if match {
					if inverse {
						continue
					}
				} else {
					if !inverse {
						continue
					}
				}

				filteredPolicies = append(filteredPolicies, policy)
			}

			switch output {
			case "table":
				rows := [][]string{}

				policies := rbac.NewSubjectPermissionsList(filteredPolicies)

				for _, p := range policies {

					var subject string
					if p.Subject.Kind == "ServiceAccount" {
						subject = fmt.Sprintf("%v/%v", p.Subject.Namespace, p.Subject.Name)
					} else {
						subject = p.Subject.Name
					}

					for _, allowedTo := range p.AllowedTo {
						row := []string{
							p.Kind,
							subject,
							allowedTo.Verb,
							allowedTo.Namespace,
							allowedTo.APIGroup,
							allowedTo.Resource,
							strings.Join(allowedTo.ResourceNames, ","),
							strings.Join(allowedTo.NonResourceURLs, ","),
							renderOriginatedFromColumn(allowedTo.Namespace, allowedTo.OriginatedFrom),
						}
						rows = append(rows, row)
					}
				}

				sort.Slice(rows, func(i, j int) bool {

					for c := range [6]int{} {
						if strings.Compare(rows[i][c], rows[j][c]) == 0 {
							continue
						}
						return (strings.Compare(rows[i][c], rows[j][c]) < 0)
					}

					return true
				})

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"TYPE", "SUBJECT", "VERBS", "NAMESPACE", "API GROUP", "KIND", "NAMES", "NonResourceURI", "ORIGINATED FROM"})
				table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
				table.SetBorder(false)
				table.SetAlignment(tablewriter.ALIGN_LEFT)
				//table.SetAutoMergeCells(true)

				table.AppendBulk(rows)
				table.Render()

				return nil
			case "yaml":
				policies := rbac.NewSubjectPermissionsList(filteredPolicies)
				data, err := yaml.Marshal(&policies)
				if err != nil {
					return fmt.Errorf("Processing error - %v", err)
				}
				fmt.Fprintln(os.Stdout, string(data))
				return nil

			case "json":
				policies := rbac.NewSubjectPermissionsList(filteredPolicies)

				data, err := json.Marshal(&policies)
				if err != nil {
					return fmt.Errorf("Processing error - %v", err)
				}

				fmt.Fprintln(os.Stdout, string(data))
				return nil

			default:
				return fmt.Errorf("Unsupported output format")
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterContext, "cluster-context", "", "Cluster Context .use 'kubectl config get-contexts' to list available contexts")
	flags.StringVarP(&output, "output", "o", "table", "Output type: table | json | yaml")

	flags.StringVarP(&regex, "regex", "e", "", "Specify whether run the lookup using a regex match")
	flags.BoolVarP(&inverse, "not", "n", false, "Inverse the regex matching. Use to search for users that do not match '^system:.*'")
	return cmd
}

func renderOriginatedFromColumn(ns string, list []v1.RoleRef) string {
	roles := sets.NewString()
	clusterRoles := sets.NewString()
	s := bytes.NewBufferString("")

	for _, ref := range list {
		if ref.Kind == "ClusterRole" {
			clusterRoles.Insert(ref.Name)
		} else {
			roles.Insert(fmt.Sprintf("%v/%v ", ns, ref.Name))
		}
	}

	if clusterRoles.Len() > 0 {
		s.WriteString(fmt.Sprintf("ClusterRoles>>%v", strings.Join(clusterRoles.List(), ",")))
	}

	if roles.Len() > 0 {
		s.WriteString(fmt.Sprintf("Roles>>%v", strings.Join(roles.List(), ",")))
	}

	return s.String()
}
