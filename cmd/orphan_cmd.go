package cmd

import (
	"encoding/json"
	"fmt"
	Orphan "github.com/alcideio/rbac-tool/pkg/orphan"
	"github.com/alcideio/rbac-tool/pkg/utils"
	v1 "k8s.io/api/core/v1"
	"os"
	"sort"
	"strings"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewCommandOrphanServiceAccounts() *cobra.Command {

	clusterContext := ""
	output := "table"
	excludedNamespaces := ""
	includedNamespaces := ""

	// Support overrides
	cmd := &cobra.Command{
		Use:           "sa",
		Aliases:       []string{"serviceaccount", "serviceaccounts", "sas"},
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "rbac-tool orphan",
		Short:         "Shows the list of Service Accounts defined but not used",
		Long: `Shows the list of Service Accounts defined but not used

Examples:

# Show Orphan ServiceAccount 
bin/rbac-tool orphan sa --include-namespaces=somennamespace

`,
		Hidden: false,
		RunE: func(c *cobra.Command, args []string) error {
			var err error

			client, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("failed to create kubernetes client - %v", err)
			}

			od, err := Orphan.NewDiscoverer(client)
			if err != nil {
				return fmt.Errorf("failed to create orphan discoverer - %v", err)
			}

			orphanSAs, err := od.GetOrphanServiceAccounts()
			if err != nil {
				return fmt.Errorf("failed to get orphan service accounts - %v", err)
			}

			inNs, exNs := utils.GetNamespaceSets(includedNamespaces, excludedNamespaces)

			filteredOrphanSAs := []*v1.ServiceAccount{}
			for i, e := range orphanSAs {

				if utils.IsNamespaceIncluded(e.Namespace, inNs, exNs) {
					filteredOrphanSAs = append(filteredOrphanSAs, orphanSAs[i])
				}
			}

			switch output {
			case "table":
				rows := [][]string{}

				for _, e := range filteredOrphanSAs {

					row := []string{
						e.Name,
						e.Namespace,
					}
					rows = append(rows, row)
				}

				sort.Slice(rows, func(i, j int) bool {
					if strings.Compare(rows[i][1], rows[j][1]) == 0 {
						return (strings.Compare(rows[i][0], rows[j][0]) < 0)
					}

					return (strings.Compare(rows[i][1], rows[j][1]) < 0)
				})

				table := tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"SERVICE ACCOUNT", "NAMESPACE"})
				table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
				table.SetBorder(false)
				table.SetAlignment(tablewriter.ALIGN_LEFT)
				//table.SetAutoMergeCells(true)

				table.AppendBulk(rows)
				table.Render()

				return nil
			case "yaml":
				data, err := yaml.Marshal(&filteredOrphanSAs)
				if err != nil {
					return fmt.Errorf("Processing error - %v", err)
				}
				fmt.Fprintln(os.Stdout, string(data))
				return nil

			case "json":
				data, err := json.Marshal(&filteredOrphanSAs)
				if err != nil {
					return fmt.Errorf("Processing error - %v", err)
				}

				fmt.Fprintln(os.Stdout, string(data))
				return nil

			default:
				return fmt.Errorf("unsupported output format")
			}
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&clusterContext, "cluster-context", "", "Cluster Context .use 'kubectl config get-contexts' to list available contexts")
	flags.StringVarP(&output, "output", "o", "table", "Output type: table | json | yaml")
	flags.StringVar(&excludedNamespaces, "exclude-namespaces", "kube-system", "Comma-delimited list of namespaces to exclude in the analysis")
	flags.StringVar(&includedNamespaces, "include-namespaces", "*", "Comma-delimited list of namespaces to include in the analysis")

	return cmd
}

func NewCommandOrphan() *cobra.Command {
	var orphanCmd = &cobra.Command{
		Use:   "orphan",
		Short: "Show orphan resources like ServiceAccounts, Roles, etc.",
		Long:  `Show orphan resources like ServiceAccounts, Roles, etc.`,
	}

	cmds := []*cobra.Command{
		NewCommandOrphanServiceAccounts(),
	}

	orphanCmd.AddCommand(cmds...)

	return orphanCmd
}
