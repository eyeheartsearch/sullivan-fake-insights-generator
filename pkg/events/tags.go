package events

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	wr "github.com/mroth/weightedrand"
)

type TagsCollection struct {
	Name string
	Tags *Tags
}

type Tags struct {
	Chooser *wr.Chooser
}

func (t *Tags) Pick() string {
	rand.Seed(time.Now().UTC().UnixNano())
	return t.Chooser.Pick().(string)
}

func NewTags(values map[string]int) (*Tags, error) {
	var choices []wr.Choice
	for k, v := range values {
		choices = append(choices, wr.Choice{
			Item:   k,
			Weight: uint(v),
		})
	}
	chooser, err := wr.NewChooser(choices...)
	if err != nil {
		return nil, err
	}
	return &Tags{
		Chooser: chooser,
	}, nil
}

func LoadTags(fileName string) ([]TagsCollection, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var values map[string]map[string]int
	if err := json.Unmarshal(fileBytes, &values); err != nil {
		return nil, err
	}

	tagsCollection := make([]TagsCollection, 0, len(values))
	for k, v := range values {
		tags, err := NewTags(v)
		if err != nil {
			return nil, err
		}
		tagsCollection = append(tagsCollection, TagsCollection{
			Name: k,
			Tags: tags,
		})
	}
	return tagsCollection, nil
}
