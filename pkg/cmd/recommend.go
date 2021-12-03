package cmd

import (
	"github.com/spf13/cobra"

	"github.com/algolia/fake-insights-generator/pkg/recommend"
)

// NewRecommendCmd creates and returns a recommend command
func NewRecommendCmd() *cobra.Command {
	cfg := &recommend.Config{}

	cmd := &cobra.Command{
		Use: "recommend",
		RunE: func(cmd *cobra.Command, args []string) error {

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
