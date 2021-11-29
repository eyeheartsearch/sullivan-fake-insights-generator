package events

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
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
	Term            SearchTerm
	ObjectIDs       []string
	QueryID         string
	Filters         []string
	ABTestVariantID int
}

// Event is a wrapper around an event to be sent to Insights.
type Event struct {
	InsightEvent *insights.Event
	SearchEvent  *SearchEvent
}

func (e *Event) EventType() string {
	if e.InsightEvent != nil {
		return e.InsightEvent.EventType
	}
	return "search"
}

// CalculatePositionWeight calculates the probability of a click on a given position.
// The formula is based on the click distribution apogee.
// Copyright (c) 2021, Sylvain Huprelle @shuprelle
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
	searchOpts := user.GetSearchOptions(cfg)
	searchOpts = append(searchOpts, opt.HitsPerPage(cfg.HitsPerPage))

	// Need to add the `GetRankingInfo` to identify the Variant ID
	if cfg.ABTest.VariantID != 0 {
		searchOpts = append(searchOpts, opt.GetRankingInfo(true))
	}

	res, err := cfg.SearchIndex.Search(searchTerm.Term, searchOpts...)
	if err != nil {
		return nil, err
	}
	// Dynamic synonyms
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
		Term:            searchTerm,
		ObjectIDs:       objectIDs,
		QueryID:         res.QueryID,
		ABTestVariantID: res.ABTestVariantID,
	}, nil
}

func MaybeClickEvent(user *User, cfg *Config, eventName string, time time.Time, searchEvent SearchEvent) *Event {
	// Get the click through rate for this specific search term
	clickThroughRate := cfg.ClickThroughRate / 100 * searchEvent.Term.ClickThroughRate

	// Improve the click through rate if A/B test is enabled and the variant is the "good" one.
	if searchEvent.ABTestVariantID != 0 && searchEvent.ABTestVariantID == cfg.ABTest.VariantID {
		clickThroughRate = clickThroughRate + cfg.ABTest.ClickThroughRate/100
	}

	if rand.Float64() > clickThroughRate {
		return nil
	}

	// Pick a random object ID to click on.
	position, err := searchEvent.PickObjectIDPosition()
	if err != nil {
		return nil
	}
	objectID := searchEvent.ObjectIDs[position]

	insightsEvent := &insights.Event{
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

	return &Event{
		InsightEvent: insightsEvent,
		SearchEvent:  &searchEvent,
	}
}

func MaybeConversionEvent(user *User, cfg *Config, eventName string, time time.Time, searchEvent SearchEvent) *Event {
	// Get the conversion rate for this specific search term
	conversionRate := cfg.ConversionRate / 100 * searchEvent.Term.ConversionRate

	// Improve the conversion rate if A/B test is enabled and the variant is the "good" one.
	if searchEvent.ABTestVariantID != 0 && searchEvent.ABTestVariantID == cfg.ABTest.VariantID {
		conversionRate = conversionRate + cfg.ABTest.ConversionRate/100
	}

	if rand.Float64() > conversionRate {
		return nil
	}

	// Pick a random object ID to convert.
	position, err := searchEvent.PickObjectIDPosition()
	if err != nil {
		return nil
	}
	objectID := searchEvent.ObjectIDs[position]

	insightsEvent := &insights.Event{
		EventType: insights.EventTypeConversion,
		EventName: eventName,
		Index:     cfg.SearchIndex.GetName(),
		UserToken: user.Token,
		Timestamp: time,
		ObjectIDs: []string{objectID},
		QueryID:   searchEvent.QueryID,
		Filters:   searchEvent.Filters,
	}

	return &Event{
		InsightEvent: insightsEvent,
		SearchEvent:  &searchEvent,
	}
}

func GenerateEventsForAllUsers(wg *sync.WaitGroup, cfg *Config, users <-chan *User) <-chan Event {
	events := make(chan Event)
	go func() {
		for user := range users {
			go GenerateEvents(wg, cfg, user, events)
		}
		wg.Wait()
		close(events)
	}()
	return events
}

func GenerateEvents(wg *sync.WaitGroup, cfg *Config, user *User, events chan<- Event) {
	defer wg.Done()
	for i := 0; i < cfg.SearchesPerUser; i++ {
		searchEvent, err := NewSearchEvent(cfg, user)
		if err != nil {
			fmt.Printf("Error doing search: %v\n", err)
			continue
		}
		events <- Event{SearchEvent: searchEvent}
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
		time.Sleep(time.Second * 2)
	}
}

func Run(cfg *Config) (StatsPerTermList, error) {
	var wg sync.WaitGroup
	users := GenerateUsers(&wg, cfg)

	eventsList := make([]Event, 0)
	for event := range GenerateEventsForAllUsers(&wg, cfg, users) {
		eventsList = append(eventsList, event)
	}

	stats := make(StatsPerTermList, 0)
	stats = append(stats, NewStatsForTerm("ALL", eventsList))
	for _, term := range cfg.SearchTerms.SearchTerms {
		stats = append(stats, NewStatsForTerm(term.Term, eventsList))
	}

	sort.Sort(stats)

	if cfg.DryRun {
		return stats, nil
	}

	// Send events to Insights
	var insightsEvent []insights.Event
	for _, event := range eventsList {
		if event.InsightEvent != nil {
			insightsEvent = append(insightsEvent, *event.InsightEvent)
		}
	}
	err := SendEvents(cfg, insightsEvent)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
