package function

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/personalization"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
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

func RestoreSettingsAndPersoStrategy(ctx context.Context, m PubSubMessage) error {
	appId := os.Getenv("ALGOLIA_APPLICATION_ID")
	apiKey := os.Getenv("ALGOLIA_API_KEY")
	indexName := os.Getenv("ALGOLIA_INDEX_NAME")

	client := search.NewClient(appId, apiKey)
	index := client.InitIndex(indexName)

	// Restore the settings
	settingsFile := os.Getenv("ECOM_SETTINGS")
	settings, err := os.Open(settingsFile)
	if err != nil {
		return err
	}
	byteValue, _ := ioutil.ReadAll(settings)
	var searchSettings search.Settings
	err = json.Unmarshal(byteValue, &searchSettings)
	if err != nil {
		return err
	}

	_, err = index.SetSettings(searchSettings)
	if err != nil {
		return errors.New("Error while restoring settings: " + err.Error())
	}

	// Restore the rules
	rulesFile := os.Getenv("ECOM_RULES")
	rules, err := os.Open(rulesFile)
	if err != nil {
		return err
	}
	byteValue, _ = ioutil.ReadAll(rules)
	var rulesList []search.Rule
	err = json.Unmarshal(byteValue, &rulesList)
	if err != nil {
		return err
	}

	_, err = index.SaveRules(rulesList)
	if err != nil {
		return errors.New("Error while restoring rules: " + err.Error())
	}

	// Restore the personalization profile
	persoStrategyFile := os.Getenv("ECOM_PERSO_STRATEGY")
	persoStrategy, err := os.Open(persoStrategyFile)
	if err != nil {
		return err
	}
	byteValue, _ = ioutil.ReadAll(persoStrategy)
	var persoStrategyContent personalization.Strategy
	err = json.Unmarshal(byteValue, &persoStrategyContent)
	if err != nil {
		return errors.New("Error while restoring personalization strategy: " + err.Error())
	}

	persoClient := personalization.NewClient(appId, apiKey, "US")
	persoClient.SetPersonalizationStrategy(persoStrategyContent)

	return nil
}
