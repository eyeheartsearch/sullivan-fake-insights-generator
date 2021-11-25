package events

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
)

type Config struct {
	DryRun bool

	SearchIndex    *search.Index
	InsightsClient *insights.Client

	SearchTerms    *SearchTerms
	TagsCollection []TagsCollection

	NumberOfUsers   int
	SearchesPerUser int

	ClickThroughRate float64
	ConversionRate   float64

	ABTest string
}
