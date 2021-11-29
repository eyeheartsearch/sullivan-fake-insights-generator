package events

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/algolia/fake-insights-generator/pkg/iostreams"
)

type ABTest struct {
	VariantID        int
	ClickThroughRate float64
	ConversionRate   float64
}

type Config struct {
	IO     *iostreams.IOStreams
	DryRun bool

	SearchIndex    *search.Index
	InsightsClient *insights.Client

	SearchTerms    *SearchTerms
	TagsCollection []TagsCollection

	NumberOfUsers   int
	SearchesPerUser int

	HitsPerPage      int
	ClickThroughRate float64
	ConversionRate   float64

	ABTest ABTest
}
