package events

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/insights"
	"github.com/montanaflynn/stats"
)

type Statistics struct {
	Cfg    *Config
	Events []insights.Event
}

func (s *Statistics) TotalEvents() int {
	return len(s.Events)
}

func (s *Statistics) EventsOfType(eventType string) []insights.Event {
	var events []insights.Event
	for _, event := range s.Events {
		if event.EventType == eventType {
			events = append(events, event)
		}
	}
	return events
}

func (s *Statistics) TotalEventsOfType(eventType string) int {
	return len(s.EventsOfType(eventType))
}

func (s *Statistics) ClickPositionList() []float64 {
	var positions []float64
	for _, event := range s.EventsOfType(insights.EventTypeClick) {
		positions = append(positions, float64(event.Positions[0]))
	}
	return positions
}

func (s *Statistics) MeanClickPosition() float64 {
	mean, _ := stats.Mean(s.ClickPositionList())
	return mean
}

func (s *Statistics) MedianClickPosition() float64 {
	median, _ := stats.Median(s.ClickPositionList())
	return median
}

func (s *Statistics) TotalSearches() int {
	return s.Cfg.NumberOfUsers * s.Cfg.SearchesPerUser
}

func (s *Statistics) TotalClicks() int {
	return s.TotalEventsOfType(insights.EventTypeClick)
}

func (s *Statistics) TotalConversions() int {
	return s.TotalEventsOfType(insights.EventTypeConversion)
}

func (s *Statistics) ClickThroughRatePercent() float64 {
	return float64(s.TotalClicks()) / float64(s.TotalSearches()) * 100
}

func (s *Statistics) ConversionRatePercent() float64 {
	return float64(s.TotalConversions()) / float64(s.TotalSearches()) * 100
}
