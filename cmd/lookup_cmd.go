package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

func NewCommandLookup() *cobra.Command {

	clusterContext := ""
	regex := ""
	inverse := false

	// Support overrides
	cmd := &cobra.Command{
		Use:		 "lookup",
		Aliases: []string{"look"},
		Short:	 "RBAC Lookup by subject (user/group/serviceaccount) name",
		Long: `
A Kubernetes RBAC lookup of Roles/ClusterRoles used by a given User/ServiceAccount/Group

Examples:

# Search All Service Accounts
rbac-tool lookup -e '.*'

# Search All Service Accounts that contain myname
rbac-tool lookup -e '.*myname.*'

# Lookup System Accounts (all accounts that start with system: )
rbac-tool lookup -e '^system:.*'

# Lookup all accounts that DO NOT start with system: )
rbac-tool lookup -ne '^system:.*'

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

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"SUBJECT", "SUBJECT TYPE", "SCOPE", "NAMESPACE", "ROLE", "BINDING TYPE"})
			table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
			table.SetBorder(false)
			table.SetAlignment(tablewriter.ALIGN_LEFT)

			rows := [][]string{}
			for _, bindings := range perms.RoleBindings {
				for _, binding := range bindings {
					for _, subject := range binding.Subjects {
						match := re.MatchString(subject.Name)

						//	match		 inverse
						//	-----------------
						//	true   true  --> skip
						//	true   false  --> keep
						//	false  true  --> keep
						//	false  false  --> skip
						if match {
							if inverse {
								continue
							}
						} else {
							if !inverse {
								continue
							}
						}

						//Subject match
						roleNamespace := binding.Namespace
						if binding.RoleRef.Kind == "ClusterRole" {
							roleNamespace = ""
						}
						_, exist := perms.Roles[roleNamespace]
						if !exist {
							continue
						}

						if binding.Namespace == "" {
							row := []string{subject.Name, subject.Kind, "ClusterRole", "", binding.RoleRef.Name}
							rows = append(rows, row)
						} else if binding.Namespace != "" && roleNamespace == "" {
							row := []string{subject.Name, subject.Kind, "ClusterRole", binding.Namespace, binding.RoleRef.Name}
							rows = append(rows, row)
						} else {
							row := []string{subject.Name, subject.Kind, "Role", binding.Namespace, binding.RoleRef.Name, "RoleBinding"}
							rows = append(rows, row)
						}
					}
				}
			}
			
			// Add ClusterRoleBindings
			for _, clusterBinding := range perms.ClusterRoleBindings {
				// Append a new row field for the binding type
				row := []string{"", "", "ClusterRole", "", clusterBinding.RoleRef.Name, "ClusterRoleBinding"}
				rows = append(rows, row)
			}

		// Modify the sorting logic to consider the new "BINDING TYPE" field
			sort.Slice(rows, func(i, j int) bool {
				if strings.Compare(rows[i][0], rows[j][0]) == 0 {
					if strings.Compare(rows[i][3], rows[j][3]) == 0 {
						return (strings.Compare(rows[i][5], rows[j][5]) < 0)
					}
					return (strings.Compare(rows[i][3], rows[j][3]) < 0)
				}
				return (strings.Compare(rows[i][0], rows[j][0]) < 0)
			})

			table.AppendBulk(rows)
			table.Render()

			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterContext, "cluster-context", "", "Cluster Context .use 'kubectl config get-contexts' to list available contexts")

	flags.StringVarP(&regex, "regex", "e", "", "Specify whether run the lookup using a regex match")
	flags.BoolVarP(&inverse, "not", "n", false, "Inverse the regex matching. Use to search for users that do not match '^system:.*'")
	return cmd
}
