package function

import (
	"context"

	"github.com/algolia/fake-insights-generator/pkg/cmd"
)

// PubSubMessage is the payload of a Pub/Sub event.
// See the documentation for more details:
// https://cloud.google.com/pubsub/docs/reference/rest/v1/PubsubMessage
type PubSubMessage struct {
	Data []byte `json:"data"`
}

func PopulateInsights(ctx context.Context, m PubSubMessage) error {
	cmdEvents := cmd.NewEventsCmd()
	err := cmdEvents.Execute()
	if err != nil {
		return err
	}
	return nil
}
