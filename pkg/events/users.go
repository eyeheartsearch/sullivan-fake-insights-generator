package events

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
)

type User struct {
	Token string
	Tags  []string

	Facets []string
}

func (u *User) String() string {
	return fmt.Sprintf("User(token=%s, tags=%s, facets=%s)", u.Token, u.Tags, u.Facets)
}

func (u *User) GetSearchOptions(cfg *Config) []interface{} {
	var opts []interface{}
	if cfg.DryRun {
		opts = append(opts, opt.Analytics(false))
	} else {
		opts = append(opts, opt.UserToken(u.Token), opt.ClickAnalytics(true))
	}
	return opts
}

func NewUser(cfg *Config) *User {
	user := &User{
		Token: fmt.Sprintf("%d", rand.Int63()),
	}
	for v := range cfg.TagsCollection {
		user.Tags = append(user.Tags, cfg.TagsCollection[v].Tags.Pick())
	}
	return user
}

func GenerateUsers(wg *sync.WaitGroup, cfg *Config) <-chan *User {
	ch := make(chan *User)
	go func() {
		for i := 0; i < cfg.NumberOfUsers; i++ {
			wg.Add(1)
			ch <- NewUser(cfg)
		}
		close(ch)
	}()
	return ch
}
