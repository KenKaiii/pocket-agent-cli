package newsapi

import (
	"github.com/spf13/cobra"
	"github.com/unstablemind/pocket/internal/common/config"
	"github.com/unstablemind/pocket/pkg/output"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "newsapi",
		Aliases: []string{"news"},
		Short:   "NewsAPI commands",
	}

	cmd.AddCommand(newHeadlinesCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newSourcesCmd())

	return cmd
}

func newHeadlinesCmd() *cobra.Command {
	var country string
	var category string
	var limit int

	cmd := &cobra.Command{
		Use:   "headlines",
		Short: "Get top headlines",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("newsapi_key")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":   "not_implemented",
				"message":  "NewsAPI headlines coming soon",
				"country":  country,
				"category": category,
				"limit":    limit,
			})
		},
	}

	cmd.Flags().StringVar(&country, "country", "us", "Country code")
	cmd.Flags().StringVar(&category, "category", "", "Category: business, entertainment, health, science, sports, technology")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of articles")

	return cmd
}

func newSearchCmd() *cobra.Command {
	var sortBy string
	var limit int

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search news articles",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("newsapi_key")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":  "not_implemented",
				"message": "NewsAPI search coming soon",
				"query":   args[0],
				"sort_by": sortBy,
				"limit":   limit,
			})
		},
	}

	cmd.Flags().StringVar(&sortBy, "sort", "publishedAt", "Sort: relevancy, popularity, publishedAt")
	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of articles")

	return cmd
}

func newSourcesCmd() *cobra.Command {
	var category string
	var country string

	cmd := &cobra.Command{
		Use:   "sources",
		Short: "List news sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.MustGet("newsapi_key")
			if err != nil {
				return err
			}

			return output.Print(map[string]any{
				"status":   "not_implemented",
				"message":  "NewsAPI sources coming soon",
				"category": category,
				"country":  country,
			})
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Category filter")
	cmd.Flags().StringVar(&country, "country", "", "Country filter")

	return cmd
}
