package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  `Manage pocket configuration: API keys, defaults, etc.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Show config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			return output.Print(map[string]string{
				"path": config.Path(),
			})
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return output.Print(cfg.Redacted())
		},
	})

	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Set(args[0], args[1]); err != nil {
				return err
			}
			return output.Print(map[string]string{
				"status": "ok",
				"key":    args[0],
			})
		},
	}
	cmd.AddCommand(setCmd)

	getCmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			val, err := config.Get(args[0])
			if err != nil {
				return err
			}
			return output.Print(map[string]string{
				"key":   args[0],
				"value": val,
			})
		},
	}
	cmd.AddCommand(getCmd)

	return cmd
}
