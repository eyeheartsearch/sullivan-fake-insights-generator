package events

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
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

// PickObjectIDPosition return the click position for a given searchEvent.
// The position is picked based on the Term's click position first if defined, then on the global click position.
func (e *SearchEvent) PickObjectIDPosition(cfg *Config) (int, error) {
	var choices []wr.Choice
	for i := range e.ObjectIDs {
		clickPosition := e.Term.ClickPosition
		if clickPosition == 0 {
			clickPosition = cfg.ClickPosition
		}
		choices = append(choices, wr.Choice{
			Weight: CalculatePositionWeight(i, clickPosition),
			Item:   i,
		})
	}

	chooser, err := wr.NewChooser(choices...)
	if err != nil {
		return 0, err
	}
	return chooser.Pick().(int), nil
}

// MaybeClickEvent returns a click event if the user clicked on a object from a SearchEvent.
func MaybeClickEvent(user *User, cfg *Config, time time.Time, searchEvent SearchEvent) *Event {
	// Get the click through rate for this specific search term
	clickThroughRate := cfg.ClickThroughRate / 100
	if searchEvent.Term.ClickThroughRate != 0 {
		clickThroughRate = searchEvent.Term.ClickThroughRate / 100
	}

	// Improve the click through rate if A/B test is enabled and the variant is the "good" one.
	if searchEvent.ABTestVariantID != 0 && searchEvent.ABTestVariantID == cfg.ABTest.VariantID {
		clickThroughRate = clickThroughRate + cfg.ABTest.ClickThroughRate/100
	}

	if rand.Float64() > clickThroughRate {
		return nil
	}

	// Pick a random object ID to click on.
	position, err := searchEvent.PickObjectIDPosition(cfg)
	if err != nil {
		return nil
	}
	objectID := searchEvent.ObjectIDs[position]

	eventName, err := cfg.EventsNames.PickForType(insights.EventTypeConversion)
	if err != nil {
		return nil
	}

	insightsEvent := &insights.Event{
		EventType: insights.EventTypeClick,
		EventName: eventName,
		Index:     cfg.SearchIndex.GetName(),
		UserToken: user.Token,
		Timestamp: time,
		ObjectIDs: []string{objectID},
		Positions: []int{position + 1}, // Positions start at 1
		QueryID:   searchEvent.QueryID,
	}

	return &Event{
		InsightEvent: insightsEvent,
		SearchEvent:  &searchEvent,
	}
}

// MaybeConversionEvent returns a conversion event if the user converted on a object from a SearchEvent.
func MaybeConversionEvent(user *User, cfg *Config, time time.Time, searchEvent SearchEvent) *Event {
	// Global conversion rate
	conversionRate := cfg.ConversionRate / 100

	// Get the conversion rate for this specific search term
	if searchEvent.Term.ConversionRate != 0 {
		conversionRate = searchEvent.Term.ConversionRate / 100
	}

	// Improve the conversion rate if A/B test is enabled and the variant is the "good" one.
	if searchEvent.ABTestVariantID != 0 && searchEvent.ABTestVariantID == cfg.ABTest.VariantID {
		conversionRate = conversionRate + cfg.ABTest.ConversionRate/100
	}

	if rand.Float64() > conversionRate {
		return nil
	}

	// Pick a random object ID to convert.
	position, err := searchEvent.PickObjectIDPosition(cfg)
	if err != nil {
		return nil
	}
	objectID := searchEvent.ObjectIDs[position]

	// Pick a conversion event name.
	eventName, err := cfg.EventsNames.PickForType(insights.EventTypeConversion)
	if err != nil {
		return nil
	}

	insightsEvent := &insights.Event{
		EventType: insights.EventTypeConversion,
		EventName: eventName,
		Index:     cfg.SearchIndex.GetName(),
		UserToken: user.Token,
		Timestamp: time,
		ObjectIDs: []string{objectID},
		QueryID:   searchEvent.QueryID,
	}

	return &Event{
		InsightEvent: insightsEvent,
		SearchEvent:  &searchEvent,
	}
}

// GenerateEventsForAllUsers generates events for all users.
// We create a pool of 100 goroutines to limit the number of concurrent requests.
func GenerateEventsForAllUsers(wg *sync.WaitGroup, cfg *Config, users <-chan *User, events chan<- Event) {
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for user := range users {
				GenerateEvents(wg, cfg, user, events)
			}
		}()
	}
}

// GenerateEvents generates events for a given user.
func GenerateEvents(wg *sync.WaitGroup, cfg *Config, user *User, events chan<- Event) {
	for i := 0; i < cfg.SearchesPerUser; i++ {
		searchEvent, err := user.Search(cfg)
		if err != nil {
			fmt.Printf("Error doing search: %v\n", err)
			continue
		}
		events <- Event{SearchEvent: searchEvent}
		if len(searchEvent.ObjectIDs) == 0 {
			fmt.Printf("Warning: No results for search term: %s - filters: %v - synonyms: %v\n", searchEvent.Term.Term, searchEvent.Filters, searchEvent.Term.Synonyms)
			continue
		}

		// Generate a click event
		clickEvent := MaybeClickEvent(user, cfg, time.Now(), *searchEvent)
		if clickEvent != nil {
			events <- *clickEvent
		}

		// Generate a conversion event
		conversionEvent := MaybeConversionEvent(user, cfg, time.Now(), *searchEvent)
		if conversionEvent != nil {
			events <- *conversionEvent
		}

		// Delay the next search to avoid triggering unwanted synonyms.
		if i < cfg.SearchesPerUser-1 {
			time.Sleep(time.Duration(cfg.SearchDelay * time.Second))
		}
	}
}

// Run is the entry point to generate the events.
func Run(cfg *Config) (StatsPerTermList, error) {
	var wg sync.WaitGroup
	users := GenerateUsers(&wg, cfg)

	events := make(chan Event)
	GenerateEventsForAllUsers(&wg, cfg, users, events)

	// Wait for all goroutines to finish and close the results channel.
	go func() {
		wg.Wait()
		close(events)
	}()

	// Gather all the events created.
	eventsList := make([]Event, 0)
	for event := range events {
		eventsList = append(eventsList, event)
	}

	// Compute the stats for each search term.
	stats := make(StatsPerTermList, 0)
	stats = append(stats, NewStatsForTerm("ALL", eventsList))
	for _, term := range cfg.SearchTerms.SearchTerms {
		stats = append(stats, NewStatsForTerm(term.Term, eventsList))
	}

	sort.Sort(stats) // Sort by number of search events

	if cfg.DryRun {
		return stats, nil
	}

	// Send events to Insights API.
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
