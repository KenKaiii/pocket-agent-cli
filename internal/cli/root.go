package cli

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/cli/commands"
	"github.com/unstablemind/pocket/pkg/output"
)

var (
	outputFormat string
	verbose      bool
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "pocket",
		Short: "Universal CLI for LLM agents",
		Long:  `Pocket is an all-in-one CLI tool designed for terminal agents to access social media, APIs, email, and more.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			output.SetFormat(outputFormat)
			output.SetVerbose(verbose)
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	root.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format: json, text, table")
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Register command groups
	root.AddCommand(commands.NewCommandsCmd())
	root.AddCommand(commands.NewIntegrationsCmd())
	root.AddCommand(commands.NewSetupCmd())
	root.AddCommand(commands.NewSocialCmd())
	root.AddCommand(commands.NewCommsCmd())
	root.AddCommand(commands.NewDevCmd())
	root.AddCommand(commands.NewProductivityCmd())
	root.AddCommand(commands.NewNewsCmd())
	root.AddCommand(commands.NewKnowledgeCmd())
	root.AddCommand(commands.NewUtilityCmd())
	root.AddCommand(commands.NewAICmd())
	root.AddCommand(commands.NewConfigCmd())

	return root
}

func Execute() error {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		// Only print if not already printed by the command
		if !output.IsPrinted(err) {
			output.PrintError("command_failed", err.Error(), nil)
		}
		return err
	}
	return nil
}
