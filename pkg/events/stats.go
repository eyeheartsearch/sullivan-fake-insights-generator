package events

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/montanaflynn/stats"
)

const (
	eventTypeSearch = "search"
)

// Stats store the statistics of the events for a given search term.
type Stats struct {
	Cfg    *Config
	Term   string
	Events []Event
}

type StatsPerTerm struct {
	Total int
	Stats Stats
}

type StatsPerTermList []*StatsPerTerm

func (s StatsPerTermList) Len() int           { return len(s) }
func (s StatsPerTermList) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s StatsPerTermList) Less(i, j int) bool { return s[i].Total < s[j].Total }

func NewStatsForTerm(term string, events []Event) *StatsPerTerm {
	var eventsForTerm []Event
	for _, event := range events {
		if event.SearchEvent.Term.Term == term || term == "ALL" {
			eventsForTerm = append(eventsForTerm, event)
		}
	}

	return &StatsPerTerm{
		Stats: Stats{
			Term:   term,
			Events: eventsForTerm,
		},
		Total: len(eventsForTerm),
	}
}

func (s *Stats) EventsOfType(eventType string) []Event {
	var events []Event
	for _, event := range s.Events {
		if event.EventType() == eventType {
			events = append(events, event)
		}
	}
	return events
}

func (s *Stats) TotalEventsOfType(eventType string) int {
	return len(s.EventsOfType(eventType))
}

func (s *Stats) ClickPositionList() []float64 {
	var positions []float64
	for _, event := range s.EventsOfType(insights.EventTypeClick) {
		positions = append(positions, float64(event.InsightEvent.Positions[0]))
	}
	return positions
}

func (s *Stats) MeanClickPosition() float64 {
	mean, _ := stats.Mean(s.ClickPositionList())
	return mean
}

func (s *Stats) MedianClickPosition() float64 {
	median, _ := stats.Median(s.ClickPositionList())
	return median
}

func (s *Stats) TotalSearches() int {
	return s.TotalEventsOfType(eventTypeSearch)
}

func (s *Stats) TotalClicks() int {
	return s.TotalEventsOfType(insights.EventTypeClick)
}

func (s *Stats) TotalConversions() int {
	return s.TotalEventsOfType(insights.EventTypeConversion)
}

func (s *Stats) ClickThroughRatePercent() float64 {
	return float64(s.TotalClicks()) / float64(s.TotalSearches()) * 100
}

func (s *Stats) ConversionRatePercent() float64 {
	return float64(s.TotalConversions()) / float64(s.TotalSearches()) * 100
}
