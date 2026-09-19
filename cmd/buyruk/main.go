package main

import (
	"os"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/cli"
)

func main() {
	rootCmd := cli.NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
