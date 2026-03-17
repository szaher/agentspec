package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("agentspec %s (commit %s, built %s, lang %s, ir %s)\n", version, commit, date, langVersion, irVersion)
		},
	}
}
