package openai

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "openai",
		Aliases: []string{"oai", "gpt"},
		Short:   "OpenAI commands",
	}

	cmd.AddCommand(newChatCmd())
	cmd.AddCommand(newModelsCmd())

	return cmd
}

func newChatCmd() *cobra.Command {
	var model string
	var maxTokens int
	var temperature float64

	cmd := &cobra.Command{
		Use:   "chat [prompt]",
		Short: "Send a chat completion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("openai_key")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":      "not_implemented",
				"message":     "OpenAI chat coming soon",
				"prompt":      args[0],
				"model":       model,
				"max_tokens":  maxTokens,
				"temperature": temperature,
			})
		},
	}

	cmd.Flags().StringVarP(&model, "model", "m", "gpt-4o", "Model to use")
	cmd.Flags().IntVar(&maxTokens, "max-tokens", 1000, "Max tokens in response")
	cmd.Flags().Float64VarP(&temperature, "temperature", "t", 0.7, "Temperature (0-2)")

	return cmd
}

func newModelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "List available models",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("openai_key")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "OpenAI models coming soon",
			})
		},
	}

	return cmd
}
