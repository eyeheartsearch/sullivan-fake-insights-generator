package events

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
)

type User struct {
	Token string
	Tags  []string

	Terms   []string
	Filters Filters
}

func (u *User) String() string {
	return fmt.Sprintf("User(token=%s, tags=%s, filters=%v)", u.Token, u.Tags, u.Filters)
}

// GetSearchOptions returns the search options for the user.
func (u *User) GetSearchOptions(cfg *Config) []interface{} {
	var opts []interface{}
	if cfg.DryRun {
		opts = append(opts, opt.Analytics(false))
	} else {
		opts = append(opts, opt.UserToken(u.Token), opt.ClickAnalytics(true), opt.AnalyticsTags(u.Tags...))
	}
	return opts
}

// GetSearchFilter returns the search filter for the user.
func (u *User) GetSearchFilter(searchTerm *SearchTerm) (string, error) {
	// User don't have any predefined filters (random user case)
	if len(u.Filters) == 0 {
		filter, err := searchTerm.Filters.Pick()
		if err != nil {
			return "", err
		}
		return filter, nil
	}
	// User has predefined filters (persona case)
	return u.Filters.Pick()
}

// Search returns a SearchEvent for the user.
// If the user have his own search terms, it will be used instead of the global ones.
func (u *User) Search(cfg *Config) (*SearchEvent, error) {
	searchOpts := u.GetSearchOptions(cfg)
	searchOpts = append(searchOpts, opt.HitsPerPage(cfg.HitsPerPage))

	// Search term
	searchTerm := cfg.SearchTerms.Pick()
	if len(u.Terms) > 0 {
		searchTerm = SearchTerm{Term: u.Terms[rand.Intn(len(u.Terms))]}
	}

	// Eventual filters
	filter, err := u.GetSearchFilter(&searchTerm)
	if err != nil {
		return nil, err
	}
	if filter != "" {
		searchOpts = append(searchOpts, opt.Filters(filter))
	}

	// Need to add the `GetRankingInfo` to identify the A/B test variant ID
	if cfg.ABTest.VariantID != 0 {
		searchOpts = append(searchOpts, opt.GetRankingInfo(true))
	}

	res, err := cfg.SearchIndex.Search(searchTerm.Term, searchOpts...)
	if err != nil {
		return nil, err
	}

	// Trigger the Dynamic Synonyms by doing directly a search with one the synonym.
	// Since we won't click or convert on the first search, it should be enough to trigger the feature.
	if len(searchTerm.Synonyms) != 0 {
		res, err = cfg.SearchIndex.Search(searchTerm.PickSynonym(), searchOpts...)
		if err != nil {
			return nil, err
		}
	}

	// Store the objectIDs so we can click / convert on them later.
	objectIDs := make([]string, 0, res.NbHits)
	for _, hit := range res.Hits {
		objectIDs = append(objectIDs, hit["objectID"].(string))
	}

	return &SearchEvent{
		Term:            searchTerm,
		ObjectIDs:       objectIDs,
		QueryID:         res.QueryID,
		ABTestVariantID: res.ABTestVariantID,
		Filters:         []string{filter},
	}, nil
}

// NewUser returns a random new user.
func NewUser(cfg *Config) *User {
	user := &User{
		Token: fmt.Sprintf("%d", rand.Int63()),
	}
	for v := range cfg.TagsCollection {
		user.Tags = append(user.Tags, cfg.TagsCollection[v].Tags.Pick())
	}
	return user
}

// NewUsersFromFile returns a list of predefined users from a file.
func NewUsersFromFile(cfg *Config, fileName string) ([]*User, error) {
	users := make([]*User, 0)
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, _ := ioutil.ReadAll(file)

	if err := json.Unmarshal(bytes, &users); err != nil {
		return nil, err
	}
	return users, nil
}

// GenerateUsers generates a list of users.
func GenerateUsers(wg *sync.WaitGroup, cfg *Config) <-chan *User {
	ch := make(chan *User)
	go func() {
		for i := 0; i < cfg.NumberOfUsers; i++ {
			ch <- NewUser(cfg)
		}
		if cfg.PersonaUsers != nil {
			for _, user := range cfg.PersonaUsers {
				ch <- user
			}
		}
		close(ch)
	}()
	return ch
}
