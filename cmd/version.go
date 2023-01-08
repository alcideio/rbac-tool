package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var (
	Version = ""
	Commit  = ""
)

func NewCommandVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print rbac-tool version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stdout, "Version: "+Version+"\nCommit: "+Commit)
		},
	}
}
