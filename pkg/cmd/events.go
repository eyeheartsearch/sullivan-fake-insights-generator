package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/spf13/cobra"

	"github.com/algolia/fake-insights-generator/pkg/events"
	"github.com/algolia/fake-insights-generator/pkg/iostreams"
	"github.com/algolia/fake-insights-generator/pkg/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewEventsCmd creates and returns an events command
func NewEventsCmd() *cobra.Command {
	cfg := &events.Config{}

	cmd := &cobra.Command{
		Use:   "events",
		Short: "Generate analytics events",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return utils.InitializeConfig(cmd, "events")
		},
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
			appId := cmd.Flag("app-id").Value.String()
			apiKey := cmd.Flag("api-key").Value.String()
			indexName := cmd.Flag("index-name").Value.String()

			if appId == "" || apiKey == "" || indexName == "" {
				return fmt.Errorf("missing required flags: app-id, api-key, index-name")
			}

			searchClient := search.NewClient(appId, apiKey)

			cfg.SearchIndex = searchClient.InitIndex(indexName)
			cfg.InsightsClient = insights.NewClient(appId, apiKey)

			return runEventsCmd(cfg)
		},
	}

	cmd.Flags().String("app-id", "", "Algolia application ID")
	cmd.Flags().String("api-key", "", "Algolia API key")
	cmd.Flags().String("index-name", "", "Algolia index name")

	cmd.Flags().String("search-terms", "search-terms.csv", "searches terms file")
	cmd.Flags().String("user-tags", "user-tags.json", "users tags file")

	cmd.Flags().IntVar(&cfg.NumberOfUsers, "users", 100, "number of users")
	cmd.Flags().IntVar(&cfg.SearchesPerUser, "searches-per-user", 5, "number of searches per user")

	cmd.Flags().IntVar(&cfg.HitsPerPage, "hits-per-page", 20, "number of hits per page")
	cmd.Flags().Float64Var(&cfg.ClickThroughRate, "click-through-rate", 20, "click through rate")
	cmd.Flags().Float64Var(&cfg.ConversionRate, "conversion-rate", 10, "conversion rate")

	// cmd.Flags().StringVar(&cfg.ABTest, "ab-test-", false, "A/B Test")
	cmd.Flags().IntVar(&cfg.ABTest.VariantID, "ab-test-variant-id", 0, "A/B Test: ID of the variant to favorize")
	cmd.Flags().Float64Var(&cfg.ABTest.ClickThroughRate, "ab-test-variant-ctr", 20, "A/B Test: How much CTR +% for the selected variant")
	cmd.Flags().Float64Var(&cfg.ABTest.ConversionRate, "ab-test-variant-cvr", 20, "A/B Test: How much CTR +% for the selected variant")

	cmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false, "if false, events will not be sent and analytics will be disabled on search queries")

	return cmd
}

func runEventsCmd(cfg *events.Config) error {
	cs := cfg.IO.ColorScheme()
	if cfg.IO.IsStdoutTTY() {
		if cfg.DryRun {
			fmt.Fprintf(cfg.IO.Out, "%s Dry run is ON: Events WILL NOT be sent to Insights and analytics will be DISABLED on search queries\n", cs.WarningIcon())
		} else {
			fmt.Fprintf(cfg.IO.Out, "%s Dry run is OFF: Events WILL be sent to Insights and analytics will be ENABLED on search queries\n", cs.WarningIcon())
		}

		if cfg.ABTest.VariantID > 0 {
			fmt.Fprintf(cfg.IO.Out, "%s A/B Test is ON: %s variant will be favorized (+%.2f%% CTR / +%.2f%% CVR)\n",
				cs.WarningIcon(), cs.Bold(string(cfg.ABTest.VariantID)), cfg.ABTest.ClickThroughRate, cfg.ABTest.ConversionRate)
		}

		cfg.IO.StartProgressIndicatorWithLabel("Generating events...")
	}

	stats, err := events.Run(cfg)

	if cfg.IO.IsStdoutTTY() {
		cfg.IO.StopProgressIndicator()
	}

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
