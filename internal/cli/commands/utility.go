package commands

import (
	"github.com/spf13/cobra"

	"github.com/unstablemind/pocket/internal/utility/crypto"
	"github.com/unstablemind/pocket/internal/utility/currency"
	"github.com/unstablemind/pocket/internal/utility/dnsbench"
	"github.com/unstablemind/pocket/internal/utility/domain"
	"github.com/unstablemind/pocket/internal/utility/geocoding"
	"github.com/unstablemind/pocket/internal/utility/holidays"
	"github.com/unstablemind/pocket/internal/utility/ipinfo"
	"github.com/unstablemind/pocket/internal/utility/netdiag"
	"github.com/unstablemind/pocket/internal/utility/paste"
	"github.com/unstablemind/pocket/internal/utility/speedtest"
	"github.com/unstablemind/pocket/internal/utility/stocks"
	"github.com/unstablemind/pocket/internal/utility/timezone"
	"github.com/unstablemind/pocket/internal/utility/traceroute"
	"github.com/unstablemind/pocket/internal/utility/translate"
	"github.com/unstablemind/pocket/internal/utility/urlshort"
	"github.com/unstablemind/pocket/internal/utility/video"
	"github.com/unstablemind/pocket/internal/utility/wayback"
	"github.com/unstablemind/pocket/internal/utility/weather"
	"github.com/unstablemind/pocket/internal/utility/wifi"
)

func NewUtilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "utility",
		Aliases: []string{"u", "util"},
		Short:   "Utility commands",
		Long:    `Utility tools: weather, crypto, stocks, currency, DNS/WHOIS, translation, etc.`,
	}

	cmd.AddCommand(weather.NewCmd())
	cmd.AddCommand(crypto.NewCmd())
	cmd.AddCommand(ipinfo.NewCmd())
	cmd.AddCommand(domain.NewCmd())
	cmd.AddCommand(currency.NewCmd())
	cmd.AddCommand(wayback.NewCmd())
	cmd.AddCommand(holidays.NewCmd())
	cmd.AddCommand(translate.NewCmd())
	cmd.AddCommand(stocks.NewCmd())
	cmd.AddCommand(urlshort.NewCmd())
	cmd.AddCommand(geocoding.NewCmd())
	cmd.AddCommand(netdiag.NewCmd())
	cmd.AddCommand(paste.NewCmd())
	cmd.AddCommand(timezone.NewCmd())
	cmd.AddCommand(speedtest.NewCmd())
	cmd.AddCommand(dnsbench.NewCmd())
	cmd.AddCommand(traceroute.NewCmd())
	cmd.AddCommand(wifi.NewCmd())
	cmd.AddCommand(video.NewCmd())

	return cmd
}
