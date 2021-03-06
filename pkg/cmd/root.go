package cmd

import (
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "fig",
		Short: "Generate search traffic and insights events",
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}

	rootCmd.AddCommand(NewEventsCmd())
	rootCmd.AddCommand(NewRecommendCmd())

	return rootCmd
}
