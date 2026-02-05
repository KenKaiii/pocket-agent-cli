package anthropic

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "anthropic",
		Aliases: []string{"claude"},
		Short:   "Anthropic commands",
	}

	cmd.AddCommand(newChatCmd())

	return cmd
}

func newChatCmd() *cobra.Command {
	var model string
	var maxTokens int

	cmd := &cobra.Command{
		Use:   "chat [prompt]",
		Short: "Send a message to Claude",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("anthropic_key")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":     "not_implemented",
				"message":    "Anthropic chat coming soon",
				"prompt":     args[0],
				"model":      model,
				"max_tokens": maxTokens,
			})
		},
	}

	cmd.Flags().StringVarP(&model, "model", "m", "claude-sonnet-4-20250514", "Model to use")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 1000, "Max tokens in response")

	return cmd
}
