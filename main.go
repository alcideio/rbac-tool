package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/alcideio/rbac-tool/cmd"
	"github.com/spf13/cobra"
)

func RbacGenCmd() *cobra.Command {
	var RootCmd = &cobra.Command{
		Use:   "rbac-tool",
		Short: "rbac-tool",
		Long:  `rbac-tool`,
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
		cmd.NewCommandVersion(),
		genBashCompletionCmd,
		cmd.NewCommandGenerateClusterRole(),
		cmd.NewCommandVisualize(),
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
