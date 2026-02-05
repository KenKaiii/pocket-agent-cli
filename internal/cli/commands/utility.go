package commands

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/utility/crypto"
	"github.com/unstablemind/pocket/internal/utility/ipinfo"
	"github.com/unstablemind/pocket/internal/utility/weather"
)

func NewUtilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "utility",
		Aliases: []string{"u", "util"},
		Short:   "Utility commands",
		Long:    `Utility tools: weather, crypto prices, IP lookup, etc.`,
	}

	cmd.AddCommand(weather.NewCmd())
	cmd.AddCommand(crypto.NewCmd())
	cmd.AddCommand(ipinfo.NewCmd())

	return cmd
}
