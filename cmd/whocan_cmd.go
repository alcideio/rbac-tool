package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"

	"github.com/antonmedv/expr"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"
)

type whoCanQuery struct {
	Verb           string
	APIGroup       string
	Kind           string
	Name           string
	NonResourceUrl string

	Rules []rbac.SubjectPolicyList
}

func NewCommandWhoCan() *cobra.Command {

	clusterContext := ""

	output := "table"
	// Support overrides
	cmd := &cobra.Command{
		Use:           "who-can",
		Aliases:       []string{"who", "whocan"},
		Args:          cobra.ExactArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "rbac-tool who-can delete deployments.apps",
		Short:         "Shows which subjects have RBAC permissions to perform an action",
		Long: `
Shows which subjects have RBAC permissions to perform an action denoted by VERB on an object denoted as ( KIND | KIND/NAME | NON-RESOURCE-URL)

* VERB is a logical Kubernetes API verb like 'get', 'list', 'watch', 'delete', etc.
* KIND is a Kubernetes resource kind. Shortcuts and API groups will be resolved, e.g. 'po' or 'deploy'.
* NAME is the name of a particular Kubernetes resource.
* NON-RESOURCE-URL is a URL that starts with "/".

Shows which subjects have RBAC permissions to <VERB>  ( KIND> | KIND/NAME | NON-RESOURCE-URL)

Examples:

# Who can read ConfigMap resources
rbac-tool who-can get cm

# Who can watch Deployments
rbac-tool who-can watch deployments.apps

# Who can read the Kubernetes API endpoint /apis
rbac-tool who-can get /apis

# Who can read a secret resource by the name some-secret
rbac-tool who-can get secret/some-secret

`,
		Hidden: false,
		RunE: func(c *cobra.Command, args []string) error {
			var err error

			kind := ""

			queryEnv := whoCanQuery{
				Verb:           args[0],
				APIGroup:       "core",
				Kind:           "*",
				Name:           "*",
				NonResourceUrl: "",
				Rules:          nil,
			}

			if len(args) == 2 {
				kind = args[1]
			}

			query := `
				filter(
					Rules, 
					{any(
						.AllowedTo, 
						 { 	.Verb     in [Verb, "*"] and 
							.Resource in [Kind, "*"] and 
							.APIGroup in [APIGroup, "*"] and 
							(Name == "*" or len(.ResourceNames) == 0 or Name in .ResourceNames)
						 }
					)}
				)`

			if strings.HasPrefix(kind, "/") {
				queryEnv.NonResourceUrl = kind
				query = `
					filter(
						Rules, 
						{any(
							.AllowedTo, 
							 { 	.Verb           in [Verb, "*"] and 
								(NonResourceUrl in .NonResourceURLs or (len(.NonResourceURLs) == 1 and .NonResourceURLs[0] == "*"))
							 }
						)}
					)`
			} else if strings.Contains(kind, "/") {
				parts := strings.Split(kind, "/")

				queryEnv.Kind = parts[0]
				queryEnv.Name = parts[1]
			} else {
				queryEnv.Kind = kind
			}

			client, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("Failed to create kubernetes client - %v", err)
			}

			if queryEnv.NonResourceUrl == "" {
				gr, err := client.Resolve(queryEnv.Verb, queryEnv.Kind, "")
				if err != nil {
					return err
				}

				queryEnv.Kind = gr.Resource
				if gr.Group != "" {
					queryEnv.APIGroup = gr.Group
				}
			}

			klog.V(8).Infof("query\n%v\n%#v\n", query, queryEnv)

			program, err := expr.Compile(query)
			if err != nil {
				return err
			}

			perms, err := rbac.NewPermissionsFromCluster(client, true)
			if err != nil {
				return err
			}

			permsPerSubject := rbac.NewSubjectPermissions(perms)
			policies := rbac.NewSubjectPermissionsList(permsPerSubject)

			queryEnv.Rules = policies

			out, err := expr.Run(program, queryEnv)
			if err != nil {
				return fmt.Errorf("Failed to run program - %v", err)
			}

			filteredPolicies := out.([]interface{})

			switch output {
			case "table":
				rows := [][]string{}

				for _, e := range filteredPolicies {
					p := e.(rbac.SubjectPolicyList)
					row := []string{
						p.Kind,
						p.Name,
						p.Namespace,
					}
					rows = append(rows, row)
				}

				sort.Slice(rows, func(i, j int) bool {
					if strings.Compare(rows[i][0], rows[j][0]) == 0 {
						return (strings.Compare(rows[i][1], rows[j][1]) < 0)
					}

					return (strings.Compare(rows[i][0], rows[j][0]) < 0)
				})

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"TYPE", "SUBJECT", "NAMESPACE"})
				table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
				table.SetBorder(false)
				table.SetAlignment(tablewriter.ALIGN_LEFT)
				//table.SetAutoMergeCells(true)

				table.AppendBulk(rows)
				table.Render()

				return nil
			case "yaml":
				data, err := yaml.Marshal(&filteredPolicies)
				if err != nil {
					return fmt.Errorf("Processing error - %v", err)
				}
				fmt.Println(string(data))
				return nil

			case "json":
				data, err := json.Marshal(&filteredPolicies)
				if err != nil {
					return fmt.Errorf("Processing error - %v", err)
				}

				fmt.Println(string(data))
				return nil

			default:
				return fmt.Errorf("Unsupported output format")
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterContext, "cluster-context", "", "Cluster Context .use 'kubectl config get-contexts' to list available contexts")
	flags.StringVarP(&output, "output", "o", "table", "Output type: table | json | yaml")

	return cmd
}
