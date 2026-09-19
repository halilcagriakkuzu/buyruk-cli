package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/halilcagriakkuzu/buyruk-cli/internal/config"
	"github.com/halilcagriakkuzu/buyruk-cli/internal/storage"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates and returns the config command.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "View and modify buyruk configuration settings",
	}

	cmd.AddCommand(NewConfigGetCmd())
	cmd.AddCommand(NewConfigSetCmd())
	cmd.AddCommand(NewConfigListCmd())

	return cmd
}

// NewConfigGetCmd creates and returns the config get command.
func NewConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Long:  "Get the value of a specific configuration key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			return getConfig(key, cmd)
		},
	}

	return cmd
}

// NewConfigSetCmd creates and returns the config set command.
func NewConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  "Set the value of a configuration key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]
			return setConfig(key, value, cmd)
		},
	}

	return cmd
}

// NewConfigListCmd creates and returns the config list command.
func NewConfigListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all configuration",
		Long:  "Display all current configuration values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listConfig(cmd)
		},
	}

	return cmd
}

// getConfig gets a configuration value and displays it.
func getConfig(key string, cmd *cobra.Command) error {
	value, err := config.GetValue(key)
	if err != nil {
		return fmt.Errorf("cli: failed to get config value: %w", err)
	}

	out := cmd.OutOrStdout()

	// Use format from command to determine output format
	format := config.ResolveFormat(cmd)

	switch format {
	case "json":
		result := map[string]string{key: value}
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("cli: failed to encode JSON: %w", err)
		}
	case "lson":
		fmt.Fprintf(out, "@%s: %s\n", strings.ToUpper(key), value)
	default: // modern
		if value == "" {
			fmt.Fprintf(out, "%s: (not set)\n", key)
		} else {
			fmt.Fprintf(out, "%s: %s\n", key, value)
		}
	}

	return nil
}

// setConfig sets a configuration value.
func setConfig(key, value string, cmd *cobra.Command) error {
	// Set config value (config.Set() handles all validation)
	if err := config.Set(key, value); err != nil {
		return fmt.Errorf("cli: failed to set config: %w", err)
	}

	// CLI-specific: warn if setting default_project to non-existent project
	if key == "default_project" && value != "" {
		projectDir, err := storage.ProjectDir(value)
		if err == nil {
			if _, err := os.Stat(projectDir); os.IsNotExist(err) {
				errOut := cmd.ErrOrStderr()
				fmt.Fprintf(errOut, "Warning: project %q does not exist\n", value)
			}
		}
	}

	// Success message
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Set %s = %s\n", key, value)

	return nil
}

// listConfig lists all configuration values.
func listConfig(cmd *cobra.Command) error {
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("cli: failed to load config: %w", err)
	}

	out := cmd.OutOrStdout()
	format := config.ResolveFormat(cmd)

	switch format {
	case "json":
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("cli: failed to encode JSON: %w", err)
		}
	case "lson":
		if cfg.DefaultProject != "" {
			fmt.Fprintf(out, "@DEFAULT_PROJECT: %s\n", cfg.DefaultProject)
		}
		if cfg.DefaultFormat != "" {
			fmt.Fprintf(out, "@DEFAULT_FORMAT: %s\n", cfg.DefaultFormat)
		}
	default: // modern
		// Use table for modern format
		table := tablewriter.NewWriter(out)
		table.SetHeader([]string{"Key", "Value"})
		table.SetBorder(false)
		table.SetColumnSeparator(" ")
		table.SetRowSeparator("")
		table.SetCenterSeparator("")

		if cfg.DefaultProject != "" {
			table.Append([]string{"default_project", cfg.DefaultProject})
		} else {
			table.Append([]string{"default_project", "(not set)"})
		}

		if cfg.DefaultFormat != "" {
			table.Append([]string{"default_format", cfg.DefaultFormat})
		} else {
			table.Append([]string{"default_format", "modern"})
		}

		table.Render()
	}

	return nil
}
