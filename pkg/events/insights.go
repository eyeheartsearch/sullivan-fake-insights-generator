package events

import "github.com/algolia/algoliasearch-client-go/v3/algolia/insights"

// Send events to Insights API in batches.
func SendEvents(i *insights.Client, events []insights.Event) error {
	chunkSize := 1000
	var chunks [][]insights.Event
	for i := 0; i < len(events); i += chunkSize {
		end := i + chunkSize

		if end > len(events) {
			end = len(events)
		}

		chunks = append(chunks, events[i:end])
	}
	for _, chunk := range chunks {
		if _, err := i.SendEvents(chunk); err != nil {
			return err
		}
	}
	return nil
}
