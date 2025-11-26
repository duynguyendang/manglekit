package main

import (
	"fmt"
	"os"

	"github.com/duynguyendang/manglekit/cmd/mkit/commands/gen"
	"github.com/duynguyendang/manglekit/cmd/mkit/commands/inspect"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mkit",
	Short: "mkit: The CLI for Neuro-Symbolic AI Governance",
	Long:  `mkit is a powerful CLI tool designed for the Manglekit framework. It facilitates neuro-symbolic AI governance by providing utilities to generate rules, inspect data schemas, and run local pipelines.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action when no subcommand is provided
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	gen.AddCommands(rootCmd)
	inspect.AddCommands(rootCmd)
	Execute()
}
