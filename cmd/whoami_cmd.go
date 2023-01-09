package cmd

import (
	"fmt"
	"os"

	"github.com/alcideio/rbac-tool/pkg/kube"
	"github.com/alcideio/rbac-tool/pkg/rbac"
	"github.com/alcideio/rbac-tool/pkg/whoami"
	"github.com/kylelemons/godebug/pretty"
	"github.com/spf13/cobra"
)

type whoAmI struct {
	Verb           string
	APIGroup       string
	Kind           string
	Name           string
	NonResourceUrl string

	Rules []rbac.SubjectPolicyList
}

func NewCommandWhoAmI() *cobra.Command {
	clusterContext := ""

	cmd := &cobra.Command{
		Use:     "whoami",
		Aliases: []string{"who-am-i"},
		Example: "rbac-tool whoami",
		Short:   "Shows the subject for the current context with which one authenticates with the cluster",
		Long:    `Shows the subject for the current context with which one authenticates with the cluster`,
		Hidden:  false,
		RunE: func(c *cobra.Command, args []string) error {
			var err error

			kubeClient, err := kube.NewClient(clusterContext)
			if err != nil {
				return fmt.Errorf("Failed to create kubernetes client - %v", err)
			}

			userInfo, err := whoami.ExtractUserInfo(kubeClient)
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stdout, pretty.Sprint(userInfo))

			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&clusterContext, "cluster-context", "c", "", "Cluster Context .use 'kubectl config get-contexts' to list available contexts")

	return cmd
}
