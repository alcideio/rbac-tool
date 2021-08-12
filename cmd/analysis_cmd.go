package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/alcideio/rbac-tool/pkg/analysis"
	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewCommandAnalysis() *cobra.Command {

	clusterContext := ""
	customConfig := ""
	output := "table"

	// Support overrides
	cmd := &cobra.Command{
		Use:           "analysis",
		Aliases:       []string{"analyze", "analyze-cluster", "an", "assess"},
		Args:          cobra.ExactArgs(0),
		SilenceUsage:  true,
		SilenceErrors: true,
		Example:       "rbac-tool analyze",
		Short:         "Analyze RBAC permissions and highlight overly permissive principals, risky permissions, etc.",
		Long: `

Examples:

# Analyze RBAC permissions of the cluster pointed by current context
rbac-tool analyze

`,
		Hidden: false,
		RunE: func(c *cobra.Command, args []string) error {
			var err error

			analysisConfig := analysis.DefaultAnalysisConfig()

			//Override Rules (if provided)
			if customConfig != "" {
				analysisConfig, err = analysis.LoadAnalysisConfig(customConfig)
				if err != nil {
					return err
				}
			}

			client, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("Failed to create kubernetes client - %v", err)
			}

			perms, err := rbac.NewPermissionsFromCluster(client)
			if err != nil {
				return err
			}

			permsPerSubject := rbac.NewSubjectPermissions(perms)
			policies := rbac.NewSubjectPermissionsList(permsPerSubject)

			analyzer := analysis.CreateAnalyzer(analysisConfig, policies)
			if analyzer == nil {
				return fmt.Errorf("Failed to create analyzer")
			}

			report, err := analyzer.Analyze()
			if err != nil {
				return err
			}

			switch output {
			//case "table":
			//	rows := [][]string{}
			//
			//	for _, e := range filteredPolicies {
			//		p := e.(rbac.SubjectPolicyList)
			//		row := []string{
			//			p.Kind,
			//			p.Name,
			//			p.Namespace,
			//		}
			//		rows = append(rows, row)
			//	}
			//
			//	sort.Slice(rows, func(i, j int) bool {
			//		if strings.Compare(rows[i][0], rows[j][0]) == 0 {
			//			return (strings.Compare(rows[i][1], rows[j][1]) < 0)
			//		}
			//
			//		return (strings.Compare(rows[i][0], rows[j][0]) < 0)
			//	})
			//
			//	table := tablewriter.NewWriter(os.Stdout)
			//	table.SetHeader([]string{"TYPE", "SUBJECT", "NAMESPACE"})
			//	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
			//	table.SetBorder(false)
			//	table.SetAlignment(tablewriter.ALIGN_LEFT)
			//	//table.SetAutoMergeCells(true)
			//
			//	table.AppendBulk(rows)
			//	table.Render()
			//
			//	return nil
			case "yaml":
				data, err := yaml.Marshal(report)
				if err != nil {
					return fmt.Errorf("Processing error - %v", err)
				}
				fmt.Println(string(data))
				return nil

			case "json":
				data, err := json.Marshal(report)
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
	flags.StringVarP(&customConfig, "config", "c", "", "Load custom analysis customConfig")

	flags.StringVar(&clusterContext, "cluster-context", "", "Cluster Context .use 'kubectl config get-contexts' to list available contexts")
	flags.StringVarP(&output, "output", "o", "yaml", "Output type: table | json | yaml")

	cmd.AddCommand(
		NewCommandGenerateAnalysisConfig(),
	)

	return cmd
}

func NewCommandGenerateAnalysisConfig() *cobra.Command {
	return &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Hidden:  true,
		Short:   "Generate Analysis Config",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := analysis.ExportDefaultConfig("yaml")
			if err != nil {
				return err
			}

			fmt.Println(c)
			return nil
		},
	}
}
