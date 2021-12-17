package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/spf13/cobra"

	"github.com/algolia/fake-insights-generator/pkg/iostreams"
	"github.com/algolia/fake-insights-generator/pkg/recommend"
	"github.com/algolia/fake-insights-generator/pkg/utils"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewRecommendCmd creates and returns a recommend command
func NewRecommendCmd() *cobra.Command {
	cfg := &recommend.Config{}

	cmd := &cobra.Command{
		Use:   "recommend",
		Short: "Generate analytics events for recommend models",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return utils.InitializeConfig(cmd, "recommend")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.IO = iostreams.System()

			// Algolia client
			appId := cmd.Flag("app-id").Value.String()
			apiKey := cmd.Flag("api-key").Value.String()
			indexName := cmd.Flag("index-name").Value.String()

			if appId == "" || apiKey == "" || indexName == "" {
				return fmt.Errorf("missing required flags: app-id, api-key, index-name")
			}

			searchClient := search.NewClient(appId, apiKey)

			cfg.SearchIndex = searchClient.InitIndex(indexName)
			cfg.InsightsClient = insights.NewClient(appId, apiKey)

			return runRecommendCmd(cfg)
		},
	}

	cmd.Flags().String("app-id", "", "Algolia application ID")
	cmd.Flags().String("api-key", "", "Algolia API key")
	cmd.Flags().String("index-name", "", "Algolia index name")

	return cmd
}

func runRecommendCmd(cfg *recommend.Config) error {
	return recommend.Run(cfg)
}
