package events

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	wr "github.com/mroth/weightedrand"
)

type SearchTerms struct {
	SearchTerms []SearchTerm
	Chooser     *wr.Chooser
}

func (t *SearchTerms) NewChooser() error {
	choices := make([]wr.Choice, 0, len(t.SearchTerms))
	weight := 20
	for _, v := range t.SearchTerms {
		if v.NoResults {
			choices = append(choices, wr.Choice{
				Item:   v,
				Weight: uint(1),
			})
		} else {
			choices = append(choices, wr.Choice{
				Item:   v,
				Weight: uint(weight),
			})
		}
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
	return t.Chooser.Pick().(SearchTerm)
}

type Filters map[string]map[string]int

func (f Filters) Pick() (string, error) {
	if len(f) == 0 {
		return "", nil
	}

	filters := make([]string, 0)
	for filterName, values := range f {
		choices := make([]wr.Choice, 0)
		for k, w := range values {
			choices = append(choices, wr.Choice{
				Item:   fmt.Sprintf("%s:\"%s\"", filterName, k),
				Weight: uint(w),
			})
		}
		chooser, err := wr.NewChooser(choices...)
		if err != nil {
			return "", err
		}
		filters = append(filters, chooser.Pick().(string))
	}
	return strings.Join(filters, " AND "), nil
}

type SearchTerm struct {
	Term             string   `json:"term"`
	ClickThroughRate float64  `json:"click_through_rate,omitempty"`
	ConversionRate   float64  `json:"conversion_rate,omitempty"`
	ClickPosition    int      `json:"click_position,omitempty"`
	Synonyms         []string `json:"synonyms,omitempty"`
	Filters          Filters  `json:"filters,omitempty"`
	NoResults        bool     `json:"no_results,omitempty"`
}

func (t *SearchTerm) PickSynonym() string {
	if len(t.Synonyms) == 0 {
		return ""
	}
	return t.Synonyms[rand.Intn(len(t.Synonyms))]
}

func NewSearchTerms(fileName string) (*SearchTerms, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, _ := ioutil.ReadAll(file)

	searchTerms := &SearchTerms{}
	if err := json.Unmarshal(bytes, &searchTerms.SearchTerms); err != nil {
		return nil, err
	}

	if err := searchTerms.NewChooser(); err != nil {
		return nil, err
	}
	return searchTerms, nil
}
