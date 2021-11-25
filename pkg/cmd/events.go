package cmd

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"text/tabwriter"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/algolia/flagship-analytics/pkg/events"
	"github.com/montanaflynn/stats"
	"github.com/spf13/cobra"
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
			cfg.InsightsClient = insights.NewClient(appId, indexName)

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
	fmt.Println("Generating events...")
	var wg sync.WaitGroup
	users := events.GenerateUsers(&wg, cfg)

	var eventsList = []insights.Event{}
	for event := range events.GenerateEventsForAllUsers(&wg, cfg, users) {
		eventsList = append(eventsList, event)
	}

	fmt.Println("Done generating events.")
	if cfg.DryRun {
		fmt.Println("Dry run is ON, not sending events to Insights.")
	} else {
		fmt.Println("Dry run is OFF, Sending events to Insights...")
		chunkSize := 1000
		var chunks [][]insights.Event
		for i := 0; i < len(eventsList); i += chunkSize {
			end := i + chunkSize

			if end > len(eventsList) {
				end = len(eventsList)
			}

			chunks = append(chunks, eventsList[i:end])
		}
		for _, chunk := range chunks {
			if _, err := cfg.InsightsClient.SendEvents(chunk); err != nil {
				return err
			}
		}
		fmt.Println("Done sending events to Insights.")
	}

	// Calculate stats
	totalClickEventsCount := 0
	var clickPositions []float64
	for _, event := range eventsList {
		if event.EventType == insights.EventTypeClick {
			clickPositions = append(clickPositions, float64(event.Positions[0]))
			totalClickEventsCount++
		}
	}
	medianClickPosition, err := stats.Median(clickPositions)
	if err != nil {
		return err
	}
	averageClickPosition, err := stats.Mean(clickPositions)
	if err != nil {
		return err
	}

	totalConversionEventsCount := 0
	for _, event := range eventsList {
		if event.EventType == insights.EventTypeConversion {
			totalConversionEventsCount++
		}
	}

	totalSearchesCount := cfg.NumberOfUsers * cfg.SearchesPerUser
	clickThroughRate := float64(totalClickEventsCount) / float64(totalSearchesCount) * 100
	conversionRate := float64(totalConversionEventsCount) / float64(totalSearchesCount) * 100

	// Show stats
	fmt.Println("Stats:")
	w := tabwriter.NewWriter(os.Stdout, 0, 1, 1, ' ', 0)
	fmt.Fprintf(w, "Total searches:\t%d", totalSearchesCount)
	fmt.Fprintf(w, "\nTotal click events:\t%d", totalClickEventsCount)
	fmt.Fprintf(w, "\nAverage click position:\t%.2f", averageClickPosition)
	fmt.Fprintf(w, "\nMedian click position:\t%.2f", medianClickPosition)
	fmt.Fprintf(w, "\nClick through rate:\t%.2f%%", clickThroughRate)
	fmt.Fprintf(w, "\nTotal conversion events:\t%d", totalConversionEventsCount)
	fmt.Fprintf(w, "\nConversion rate:\t%.2f%%", conversionRate)
	w.Flush()

	return nil
}
