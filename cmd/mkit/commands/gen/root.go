package gen

import (
	"github.com/spf13/cobra"
)

var GenCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate Manglekit assets",
	Aliases: []string{"gen"},
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(GenCmd)
}
