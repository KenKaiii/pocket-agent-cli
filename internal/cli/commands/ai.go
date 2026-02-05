package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/ai/openai"
	"github.com/unstablemind/pocket/internal/ai/anthropic"
)

func NewAICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ai",
		Aliases: []string{"llm"},
		Short:   "AI and LLM commands",
		Long:    `Interact with AI services: OpenAI, Anthropic, etc.`,
	}

	cmd.AddCommand(openai.NewCmd())
	cmd.AddCommand(anthropic.NewCmd())

	return cmd
}
