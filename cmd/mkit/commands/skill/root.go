package skill

import (
	"github.com/spf13/cobra"
)

var SkillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Scaffold and manage Manglekit skills",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(SkillCmd)
	SkillCmd.AddCommand(newCmd)
}
