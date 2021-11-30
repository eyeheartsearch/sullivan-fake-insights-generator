package events

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	wr "github.com/mroth/weightedrand"
)

type EventNames map[string]map[string]int

func (n EventNames) PickForType(eventType string) (string, error) {
	if len(n) == 0 || len(n[eventType]) == 0 {
		return "", fmt.Errorf("no event names for the type \"%s\" found", eventType)
	}
	choices := make([]wr.Choice, 0, len(n))
	for k, w := range n[eventType] {
		choices = append(choices, wr.Choice{
			Item:   k,
			Weight: uint(w),
		})
	}
	chooser, err := wr.NewChooser(choices...)
	if err != nil {
		return "", err
	}
	return chooser.Pick().(string), nil
}

func EventNamesFromFile(filename string) (EventNames, error) {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var n EventNames
	if err := json.Unmarshal(b, &n); err != nil {
		return nil, err
	}
	return n, nil
}
