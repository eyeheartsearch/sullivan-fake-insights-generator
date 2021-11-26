package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/spf13/cobra"

	"github.com/algolia/fake-insights-generator/pkg/events"
	"github.com/algolia/fake-insights-generator/pkg/iostreams"
	"github.com/algolia/fake-insights-generator/pkg/utils"
)

// NewEventsCmd creates and returns an events command
func NewEventsCmd() *cobra.Command {
	cfg := &events.Config{}

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Generate analytics events",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.IO = iostreams.System()

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
	cs := cfg.IO.ColorScheme()
	if cfg.DryRun {
		fmt.Fprintf(cfg.IO.Out, "%s Dry run is ON: Events WILL NOT be sent to Insights and analytics will be DISABLED on search queries\n", cs.WarningIcon())
	} else {
		fmt.Fprintf(cfg.IO.Out, "%s Dry run is OFF: Events WILL be sent to Insights and analytics will be ENABLED on search queries\n", cs.WarningIcon())
	}

	cfg.IO.StartProgressIndicatorWithLabel("Generating events...")
	stats, err := events.Run(cfg)
	cfg.IO.StopProgressIndicator()
	if err != nil {
		return err
	}

	if cfg.IO.IsStdoutTTY() {
		fmt.Fprintf(cfg.IO.Out, "%s Done generating events!\n", cs.SuccessIcon())
	}

	table := utils.NewTablePrinter(cfg.IO)
	if table.IsTTY() {
		table.AddField(cs.Bold("TERM"), nil, nil)
		table.AddField(cs.Bold("SEARCHES"), nil, nil)
		table.AddField(cs.Bold("CLICKS"), nil, nil)
		table.AddField(cs.Bold("CLICK THROUGH RATE"), nil, nil)
		table.AddField(cs.Bold("AVG CLICK POSITION"), nil, nil)
		table.AddField(cs.Bold("MEDIAN CLICK POSITION"), nil, nil)
		table.AddField(cs.Bold("CONVERSIONS"), nil, nil)
		table.AddField(cs.Bold("CONVERSION RATE"), nil, nil)
		table.EndRow()
	}

	for _, stats := range stats {
		table.AddField(stats.Stats.Term, nil, nil)
		table.AddField(fmt.Sprintf("%d", stats.Stats.TotalSearches()), nil, nil)
		table.AddField(fmt.Sprintf("%d", stats.Stats.TotalEventsOfType(insights.EventTypeClick)), nil, nil)
		table.AddField(fmt.Sprintf("%.2f%%", stats.Stats.ClickThroughRatePercent()), nil, nil)
		table.AddField(fmt.Sprintf("%.2f", stats.Stats.MeanClickPosition()), nil, nil)
		table.AddField(fmt.Sprintf("%.2f", stats.Stats.MedianClickPosition()), nil, nil)
		table.AddField(fmt.Sprintf("%d", stats.Stats.TotalConversions()), nil, nil)
		table.AddField(fmt.Sprintf("%.2f%%", stats.Stats.ConversionRatePercent()), nil, nil)
		table.EndRow()
	}

	return table.Render()
}
