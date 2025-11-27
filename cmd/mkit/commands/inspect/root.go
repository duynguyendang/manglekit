package inspect

import (
	"github.com/spf13/cobra"
)

var InspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect Manglekit objects",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(InspectCmd)
}
