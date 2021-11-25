package cmd

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/spf13/cobra"

	"github.com/algolia/flagship-analytics/pkg/events"
)

// NewEventsCmd creates and returns an events command
func NewEventsCmd() *cobra.Command {
	cfg := &events.Config{}

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Generate analytics events",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Search terms
			searchTermsFileName := cmd.Flag("search-terms").Value.String()
			searches, err := events.NewSearchTerms(searchTermsFileName)
			if err != nil {
				return err
			}
			cfg.SearchTerms = searches

			// Users tags
			usersTagsFileName := cmd.Flag("user-tags").Value.String()
			tagsCollection, err := events.LoadTags(usersTagsFileName)
			if err != nil {
				return err
			}
			cfg.TagsCollection = tagsCollection

			// Algolia clients (search and insights)
			appId := os.Getenv("ALGOLIA_APP_ID")
			if appId == "" {
				return errors.New("ALGOLIA_APP_ID environment variable is not set")
			}
			apiKey := os.Getenv("ALGOLIA_API_KEY")
			if apiKey == "" {
				return errors.New("ALGOLIA_API_KEY environment variable is not set")
			}
			indexName := os.Getenv("ALGOLIA_INDEX_NAME")
			if indexName == "" {
				return errors.New("ALGOLIA_INDEX_NAME environment variable is not set")
			}
			searchClient := search.NewClient(appId, apiKey)

			cfg.SearchIndex = searchClient.InitIndex(indexName)
			cfg.InsightsClient = insights.NewClient(appId, apiKey)

			return runEventsCmd(cfg)
		},
	}

	cmd.Flags().String("search-terms", "search-terms.csv", "searches terms file")
	cmd.Flags().String("user-tags", "user-tags.json", "users tags file")

	cmd.Flags().IntVar(&cfg.NumberOfUsers, "users", 100, "number of users")
	cmd.Flags().IntVar(&cfg.SearchesPerUser, "searches-per-user", 5, "number of searches per user")

	cmd.Flags().IntVar(&cfg.HitsPerPage, "hits-per-page", 20, "number of hits per page")
	cmd.Flags().Float64Var(&cfg.ClickThroughRate, "click-through-rate", 20, "click through rate")
	cmd.Flags().Float64Var(&cfg.ConversionRate, "conversion-rate", 10, "conversion rate")

	// cmd.Flags().StringVar(&cfg.ABTest, "ab-test-", false, "A/B Test")

	cmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false, "if false, events will not be sent and analytics will be disabled on search queries")

	return cmd
}

func runEventsCmd(cfg *events.Config) error {
	stats, err := events.Run(cfg)
	if err != nil {
		return err
	}

	// Show stats
	fmt.Println("Stats:")
	w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', 0)
	fmt.Fprintf(w, "\nTotal searches:\t%d", stats.TotalSearches())
	fmt.Fprintf(w, "\nTotal click events:\t%d", stats.TotalEventsOfType(insights.EventTypeClick))
	fmt.Fprintf(w, "\nAverage click position:\t%.2f", stats.MeanClickPosition())
	fmt.Fprintf(w, "\nMedian click position:\t%.2f", stats.MedianClickPosition())
	fmt.Fprintf(w, "\nClick through rate:\t%.2f%%", stats.ClickThroughRatePercent())
	fmt.Fprintf(w, "\nTotal conversion events:\t%d", stats.TotalConversions())
	fmt.Fprintf(w, "\nConversion rate:\t%.2f%%\n", stats.ConversionRatePercent())
	w.Flush()

	return nil
}
