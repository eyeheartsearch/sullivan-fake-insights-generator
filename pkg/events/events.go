package events

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
	wr "github.com/mroth/weightedrand"
)

const (
	clickDistributionApogee = 20
)

type SearchEvent struct {
	Term      SearchTerm
	ObjectIDs []string
	QueryID   string
	Filters   []string
}

func CalculatePositionWeight(itemPosition int, clickPosition int) uint {
	a := float64(itemPosition - clickPosition)
	b := float64(2 * clickPosition)
	return uint(1 + clickDistributionApogee*math.Exp(-(math.Pow(a, 2)/b)))
}

func (e *SearchEvent) PickObjectIDPosition() (int, error) {
	var choices []wr.Choice
	for i := range e.ObjectIDs {
		choices = append(choices, wr.Choice{
			Weight: CalculatePositionWeight(i, e.Term.ClickPosition),
			Item:   i,
		})
	}

	chooser, err := wr.NewChooser(choices...)
	if err != nil {
		return 0, err
	}
	return chooser.Pick().(int), nil
}

func NewSearchEvent(cfg *Config, user *User) (*SearchEvent, error) {
	searchTerm := cfg.SearchTerms.Pick()
	// User specific search options
	searchOpts := user.GetSearchOptions(cfg)
	// Hits per page
	searchOpts = append(searchOpts, opt.HitsPerPage(cfg.HitsPerPage))

	res, err := cfg.SearchIndex.Search(searchTerm.Term, searchOpts...)
	if err != nil {
		return nil, err
	}
	// dynamic synonyms
	if searchTerm.Synonym != "" {
		res, err = cfg.SearchIndex.Search(searchTerm.Synonym, searchOpts...)
		if err != nil {
			return nil, err
		}
	}

	objectIDs := make([]string, 0, res.NbHits)
	for _, hit := range res.Hits {
		objectIDs = append(objectIDs, hit["objectID"].(string))
	}

	return &SearchEvent{
		Term:      searchTerm,
		ObjectIDs: objectIDs,
		QueryID:   res.QueryID,
	}, nil
}

func MaybeClickEvent(user *User, cfg *Config, eventName string, time time.Time, searchEvent SearchEvent) *insights.Event {
	// Get the click through rate for this specific search term
	clickThroughRate := cfg.ClickThroughRate / 100 * searchEvent.Term.ClickThroughRate
	if rand.Float64() > clickThroughRate {
		return nil
	}

	// Pick a random object ID to click on.
	position, err := searchEvent.PickObjectIDPosition()
	if err != nil {
		return nil
	}
	objectID := searchEvent.ObjectIDs[position]

	return &insights.Event{
		EventType: insights.EventTypeClick,
		EventName: eventName,
		Index:     cfg.SearchIndex.GetName(),
		UserToken: user.Token,
		Timestamp: time,
		ObjectIDs: []string{objectID},
		Positions: []int{position + 1}, // Positions start at 1
		QueryID:   searchEvent.QueryID,
		Filters:   searchEvent.Filters,
	}
}

func MaybeConversionEvent(user *User, cfg *Config, eventName string, time time.Time, searchEvent SearchEvent) *insights.Event {
	// Get the conversion rate for this specific search term
	conversionRate := cfg.ConversionRate / 100 * searchEvent.Term.ConversionRate
	if rand.Float64() > conversionRate {
		return nil
	}

	// Pick a random object ID to convert.
	position, err := searchEvent.PickObjectIDPosition()
	if err != nil {
		return nil
	}
	objectID := searchEvent.ObjectIDs[position]

	return &insights.Event{
		EventType: insights.EventTypeConversion,
		EventName: eventName,
		Index:     cfg.SearchIndex.GetName(),
		UserToken: user.Token,
		Timestamp: time,
		ObjectIDs: []string{objectID},
		QueryID:   searchEvent.QueryID,
		Filters:   searchEvent.Filters,
	}
}

func GenerateEventsForAllUsers(wg *sync.WaitGroup, cfg *Config, users <-chan *User) <-chan insights.Event {
	events := make(chan insights.Event)
	go func() {
		for user := range users {
			go GenerateEvents(wg, cfg, user, events)
		}
		wg.Wait()
		close(events)
	}()
	return events
}

func GenerateEvents(wg *sync.WaitGroup, cfg *Config, user *User, events chan<- insights.Event) {
	defer wg.Done()
	for i := 0; i < cfg.SearchesPerUser; i++ {
		searchEvent, err := NewSearchEvent(cfg, user)
		if err != nil {
			continue
		}
		if len(searchEvent.ObjectIDs) == 0 {
			fmt.Printf("Warning: No results for search term: %s (%s)\n", searchEvent.Term.Term, searchEvent.Term.Synonym)
			continue
		}

		// Generate a click event
		clickEvent := MaybeClickEvent(user, cfg, "click", time.Now(), *searchEvent)
		if clickEvent != nil {
			events <- *clickEvent
		}

		// Generate a conversion event
		conversionEvent := MaybeConversionEvent(user, cfg, "conversion", time.Now(), *searchEvent)
		if conversionEvent != nil {
			events <- *conversionEvent
		}
		time.Sleep(2 * time.Second)
	}
}
