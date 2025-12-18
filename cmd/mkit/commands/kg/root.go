package kg

import (
	"github.com/spf13/cobra"
)

var KgCmd = &cobra.Command{
	Use:   "kg",
	Short: "Knowledge Graph utilities",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func AddCommands(rootCmd *cobra.Command) {
	KgCmd.AddCommand(ConvertCmd)
	rootCmd.AddCommand(KgCmd)
}
