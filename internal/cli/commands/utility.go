package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/utility/weather"
)

func NewUtilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "utility",
		Aliases: []string{"u", "util"},
		Short:   "Utility commands",
		Long:    `Utility tools: weather, IP lookup, etc.`,
	}

	cmd.AddCommand(weather.NewCmd())

	return cmd
}
