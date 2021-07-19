package utils

import (
	"fmt"
	"github.com/fatih/color"
)

var rbacToolPrefix = color.New(color.FgBlue).SprintFunc()
var lineMsg = color.New(color.FgHiWhite).SprintFunc()

func ConsolePrinter(msg string) {
	fmt.Println(rbacToolPrefix("[RAPID7-INSIGHTCLOUDSEC]"), lineMsg(msg))
}
