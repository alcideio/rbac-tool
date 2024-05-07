package utils

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	rbacToolPrefix = color.New(color.FgBlue).SprintFunc()
	lineMsg        = color.New(color.FgHiWhite).SprintFunc()
)

func ConsolePrinter(msg string) {
	fmt.Fprintln(os.Stderr, rbacToolPrefix("[RAPID7-INSIGHTCLOUDSEC]"), lineMsg(msg))
}
