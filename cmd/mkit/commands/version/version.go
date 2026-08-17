// Package version implements the `mkit version` subcommand.
package version

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// VersionCmd prints the module version of the mkit binary.
var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the mkit version",
	Long: `Print the mkit version.

The version is read from the Go module build info of this binary
(runtime/debug.ReadBuildInfo). For binaries built from a working tree it
reports (devel).`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "mkit %s\n", ModuleVersion())
	},
}

// ModuleVersion returns the module version from build info, falling back
// to "(devel)" when unavailable (e.g. `go run` or `go build` from a
// non-module-tagged working tree).
func ModuleVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" {
		return bi.Main.Version
	}
	return "(devel)"
}

func init() {
	VersionCmd.SilenceUsage = true
}

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(VersionCmd)
}
