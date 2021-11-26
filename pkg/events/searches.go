package events

import (
	"math/rand"
	"os"
	"time"

	"github.com/gocarina/gocsv"
	wr "github.com/mroth/weightedrand"
)

type SearchTerms struct {
	SearchTerms []SearchTerm
	Chooser     *wr.Chooser
}

func (t *SearchTerms) NewChooser() error {
	choices := make([]wr.Choice, 0, len(t.SearchTerms))
	weight := 100
	for _, v := range t.SearchTerms {
		choices = append(choices, wr.Choice{
			Item:   v,
			Weight: uint(weight),
		})
		if weight > 1 {
			weight--
		}
	}
	chooser, err := wr.NewChooser(choices...)
	if err != nil {
		return err
	}
	t.Chooser = chooser
	return nil
}

func (t *SearchTerms) Pick() SearchTerm {
	rand.Seed(time.Now().UTC().UnixNano())
	return t.Chooser.Pick().(SearchTerm)
}

type SearchTerm struct {
	Term             string  `csv:"term"`
	ClickThroughRate float64 `csv:"click_through_rate"`
	ConversionRate   float64 `csv:"conversion_rate"`
	ClickPosition    int     `csv:"click_position"`
	Synonym          string  `csv:"synonym"`
	Facets           string  `csv:"facets"`
}

func NewSearchTerms(fileName string) (*SearchTerms, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	searchTerms := &SearchTerms{}
	if err := gocsv.UnmarshalFile(file, &searchTerms.SearchTerms); err != nil {
		return nil, err
	}
	if err := searchTerms.NewChooser(); err != nil {
		return nil, err
	}
	return searchTerms, nil
}
