package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/gadinaor/rbac-cluster-role/cmd"
	"github.com/spf13/cobra"
)

func RbacGenCmd() *cobra.Command {
	var RootCmd = &cobra.Command{
		Use:   "rbac-minimizer",
		Short: "rbac-minimizer",
		Long:  `rbac-minimizer`,
	}

	var genBashCompletionCmd = &cobra.Command{
		Use:   "bash-completion",
		Short: "Generate bash completion. source < (advisor bash-completion)",
		Long:  "Generate bash completion. source < (advisor bash-completion)",
		Run: func(cmd *cobra.Command, args []string) {
			out := new(bytes.Buffer)
			_ = RootCmd.GenBashCompletion(out)
			println(out.String())
		},
	}

	cmds := []*cobra.Command{
		genBashCompletionCmd,
		cmd.NewCommandGenerateClusterRole(),
	}

	RootCmd.AddCommand(cmds...)

	return RootCmd
}

func main() {
	rootCmd := RbacGenCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
