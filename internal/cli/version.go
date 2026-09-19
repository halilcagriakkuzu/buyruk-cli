package cli

import (
	"fmt"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/build"
	"github.com/spf13/cobra"
)

// NewVersionCmd creates and returns the version command.
func NewVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of buyruk",
		Long:  "Print the version number of buyruk CLI tool",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := GetFormat(cmd)
			out := cmd.OutOrStdout()
			switch format {
			case "json":
				fmt.Fprintf(out, `{"version":"%s"}`+"\n", build.Version)
			case "lson":
				fmt.Fprintf(out, "@VERSION: %s\n", build.Version)
			default: // modern
				fmt.Fprintf(out, "buyruk version %s\n", build.Version)
			}
			return nil
		},
	}
	return cmd
}
